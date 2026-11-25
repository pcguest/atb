// SPDX-License-Identifier: MIT

// Package strict implements the strict atb-jcs-rfc8785-1 canonicalisation
// profile: RFC 8785 plus the ATB safe-integer interoperability restriction.
//
// It is a new, non-production entry point. The legacy canonicaliser in
// internal/canonicalize, the hashing path, bundle issuance/verification, and
// manifest dispatch do not call this package.
package strict

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// Stable error classes reported by this package.
const (
	ClassDuplicateKey          = "duplicate_key"
	ClassInvalidUTF8           = "invalid_utf8"
	ClassLoneSurrogate         = "lone_surrogate"
	ClassTrailingData          = "trailing_data"
	ClassNonFiniteNumber       = "non_finite_number"
	ClassNumberOutOfPolicy     = "number_out_of_policy"
	ClassNumberUnrepresentable = "number_unrepresentable"
	ClassInvalidJSON           = "invalid_json"
	ClassDepthLimit            = "depth_limit"
)

// maxNestingDepth bounds array/object nesting in both the raw parser and the
// reflected value walk. It keeps deeply nested or self-referential inputs from
// exhausting the stack.
const maxNestingDepth = 512

// maxSafeInteger is 2^53 - 1, the largest plain-integer literal magnitude the
// strict profile accepts.
var maxSafeInteger = big.NewInt(9007199254740991)

const (
	maxSafeIntegerInt64  = int64(9007199254740991)
	maxSafeIntegerUint64 = uint64(9007199254740991)
)

// Error is a strict-profile canonicalisation failure carrying a stable class.
type Error struct {
	Class   string
	Offset  int
	Message string
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("canonicalize/strict: %s at offset %d: %s", e.Class, e.Offset, e.Message)
}

// ErrorClass returns the stable error class of err, or "" when err is nil or
// does not originate from this package.
func ErrorClass(err error) string {
	var target *Error
	if errors.As(err, &target) {
		return target.Class
	}
	return ""
}

// Canonicalize returns the strict canonical form of a single raw JSON value.
func Canonicalize(raw []byte) ([]byte, error) {
	return canonicalize(raw, false)
}

func canonicalize(raw []byte, valueMode bool) ([]byte, error) {
	if !utf8.Valid(raw) {
		return nil, &Error{
			Class:   ClassInvalidUTF8,
			Offset:  firstInvalidUTF8(raw),
			Message: "input is not valid UTF-8",
		}
	}

	p := &parser{data: raw, valueMode: valueMode}
	var buf bytes.Buffer
	if err := p.parseValue(&buf); err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.pos != len(p.data) {
		return nil, &Error{
			Class:   ClassTrailingData,
			Offset:  p.pos,
			Message: "non-whitespace data follows the top-level JSON value",
		}
	}
	return buf.Bytes(), nil
}

// CanonicalizeValue marshals v with encoding/json and then strict-canonicalises
// the result. Go integer values outside the safe-integer policy and non-finite
// floats are rejected directly, so the plain-integer token policy is not
// re-applied to the marshalled output (a finite float64 such as 1e20 marshals
// as a plain integer token but is a valid binary64 value).
func CanonicalizeValue(v any) ([]byte, error) {
	if err := validateValue(v); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(v)
	if err != nil {
		if isNonFiniteMarshalError(err) {
			return nil, &Error{
				Class:   ClassNonFiniteNumber,
				Offset:  0,
				Message: "value contains a non-finite number",
			}
		}
		return nil, err
	}
	return canonicalize(raw, true)
}

// isNonFiniteMarshalError reports whether err is encoding/json's rejection of a
// NaN or infinity float value.
func isNonFiniteMarshalError(err error) bool {
	var unsupported *json.UnsupportedValueError
	if !errors.As(err, &unsupported) {
		return false
	}
	value := unsupported.Value
	if !value.IsValid() {
		return false
	}
	switch value.Kind() {
	case reflect.Float32, reflect.Float64:
		f := value.Float()
		return math.IsNaN(f) || math.IsInf(f, 0)
	default:
		return false
	}
}

var (
	jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	jsonNumberType    = reflect.TypeOf(json.Number(""))
	rawMessageType    = reflect.TypeOf(json.RawMessage(nil))
)

// visit identifies a pointer-like value on the current traversal path. Pointer
// keys include the pointed-to type so that a pointer to a struct's first field
// is not mistaken for a cycle with a pointer to the struct itself. The length
// and capacity are part of the slice identity so that distinct views over one
// backing array are not mistaken for cycles.
type visit struct {
	kind reflect.Kind
	ptr  uintptr
	len  int
	cap  int
	typ  reflect.Type
}

// validateValue walks a Go value and enforces the strict profile's numeric
// policy: integer values must fit the safe-integer range and float values must
// be finite. Dynamic JSON fragments (json.Number, json.RawMessage and custom
// marshalers) are validated by re-running the strict parser over their bytes.
func validateValue(v any) error {
	return validateReflect(reflect.ValueOf(v), make(map[visit]bool), 0)
}

func validateReflect(v reflect.Value, seen map[visit]bool, depth int) error {
	if !v.IsValid() {
		return nil
	}
	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Type() {
	case jsonNumberType:
		return validateNumberLiteral(v.String())
	case rawMessageType:
		// json.Marshal encodes a nil RawMessage as null, while an empty
		// non-nil RawMessage is still invalid.
		if v.IsNil() {
			return nil
		}
		return validateRawFragment(v.Bytes())
	}

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		if hasCustomMarshaler(v) {
			return validateCustomMarshaler(v)
		}
		key := visit{kind: reflect.Pointer, ptr: v.Pointer(), typ: v.Type()}
		if seen[key] {
			return cyclicValueError()
		}
		// Pointer traversal counts towards the nesting bound: without this an
		// arbitrarily long pointer chain could exhaust the stack even though
		// arrays, structs, and maps are already bounded.
		next := depth + 1
		if next > maxNestingDepth {
			return depthLimitError()
		}
		seen[key] = true
		err := validateReflect(v.Elem(), seen, next)
		delete(seen, key)
		return err
	}

	if hasCustomMarshaler(v) {
		return validateCustomMarshaler(v)
	}

	switch v.Kind() {
	case reflect.String:
		if !utf8.ValidString(v.String()) {
			return &Error{
				Class:   ClassInvalidUTF8,
				Message: "string value is not valid UTF-8",
			}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		x := v.Int()
		if x > maxSafeIntegerInt64 || x < -maxSafeIntegerInt64 {
			return &Error{
				Class:   ClassNumberOutOfPolicy,
				Message: "integer value exceeds 2^53 - 1",
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if v.Uint() > maxSafeIntegerUint64 {
			return &Error{
				Class:   ClassNumberOutOfPolicy,
				Message: "integer value exceeds 2^53 - 1",
			}
		}
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return &Error{
				Class:   ClassNonFiniteNumber,
				Message: "value contains a non-finite number",
			}
		}
	case reflect.Slice:
		if v.IsNil() {
			return nil
		}
		next := depth + 1
		if next > maxNestingDepth {
			return depthLimitError()
		}
		if v.Len() > 0 {
			key := visit{kind: reflect.Slice, ptr: v.Pointer(), len: v.Len(), cap: v.Cap()}
			if seen[key] {
				return cyclicValueError()
			}
			seen[key] = true
			defer delete(seen, key)
		}
		for i := 0; i < v.Len(); i++ {
			if err := validateReflect(v.Index(i), seen, next); err != nil {
				return err
			}
		}
	case reflect.Array:
		next := depth + 1
		if next > maxNestingDepth {
			return depthLimitError()
		}
		for i := 0; i < v.Len(); i++ {
			if err := validateReflect(v.Index(i), seen, next); err != nil {
				return err
			}
		}
	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		next := depth + 1
		if next > maxNestingDepth {
			return depthLimitError()
		}
		key := visit{kind: reflect.Map, ptr: v.Pointer()}
		if seen[key] {
			return cyclicValueError()
		}
		seen[key] = true
		defer delete(seen, key)
		iter := v.MapRange()
		for iter.Next() {
			if err := validateMapKey(iter.Key()); err != nil {
				return err
			}
			if err := validateReflect(iter.Value(), seen, next); err != nil {
				return err
			}
		}
	case reflect.Struct:
		return validateStructFields(v, seen, depth)
	}
	return nil
}

// validateStructFields applies encoding/json's struct-field rules to v, which
// must be a struct.
func validateStructFields(v reflect.Value, seen map[visit]bool, depth int) error {
	next := depth + 1
	if next > maxNestingDepth {
		return depthLimitError()
	}
	structType := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := structType.Field(i)
		tag := field.Tag.Get("json")
		if tag == "-" {
			continue
		}
		embeddedStruct := isEmbeddedStructField(field)
		if field.PkgPath != "" && !embeddedStruct {
			// encoding/json ignores unexported non-embedded fields and
			// embedded fields of unexported non-struct types, but it still
			// walks the exported fields of an embedded unexported struct.
			continue
		}
		fieldValue := v.Field(i)
		// omitempty/omitzero only apply when encoding/json emits the field
		// itself; an untagged embedded struct is flattened into its parent,
		// so its own options have no effect there.
		if (!embeddedStruct || jsonTagName(tag) != "") &&
			((hasJSONOption(tag, "omitempty") && isEmptyValue(fieldValue)) ||
				(hasJSONOption(tag, "omitzero") && isZeroValue(field.Type, fieldValue))) {
			continue
		}
		if emitsQuotedValue(field, tag, fieldValue) {
			// The "string" option makes encoding/json emit a quoted JSON string
			// for bool, string, integer, and float fields, so their underlying
			// value is not a JSON number and the numeric policy does not apply.
			continue
		}
		if embeddedStruct {
			// encoding/json may flatten an embedded struct into its parent even
			// when the embedded type implements a custom marshaler (for example,
			// when a sibling method makes the parent itself non-Marshaler). It
			// may also use that marshaler directly. Validate both possibilities
			// rather than short-circuiting on the embedded field's marshaler.
			// This can over-reject when the promotion is unused, but it stays
			// fail-closed: it may never under-reject.
			if err := validateEmbeddedStructField(fieldValue, seen, next); err != nil {
				return err
			}
			continue
		}
		if err := validateReflect(fieldValue, seen, next); err != nil {
			return err
		}
	}
	return nil
}

// validateEmbeddedStructField validates an anonymous embedded struct or
// pointer-to-struct field. Because encoding/json can either invoke a custom
// marshaler promoted by the embedded type or flatten the embedded struct's
// fields into the parent, validate both the marshaler output that could be used
// and the embedded fields themselves.
func validateEmbeddedStructField(v reflect.Value, seen map[visit]bool, depth int) error {
	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			// encoding/json omits nil embedded pointer-to-struct fields.
			return nil
		}
		if hasCustomMarshaler(v) {
			if err := validateCustomMarshaler(v); err != nil {
				return err
			}
		}
		key := visit{kind: reflect.Pointer, ptr: v.Pointer(), typ: v.Type()}
		if seen[key] {
			return cyclicValueError()
		}
		next := depth + 1
		if next > maxNestingDepth {
			return depthLimitError()
		}
		seen[key] = true
		err := validateEmbeddedStructField(v.Elem(), seen, next)
		delete(seen, key)
		return err
	}
	if v.Kind() != reflect.Struct {
		return validateReflect(v, seen, depth)
	}
	if hasCustomMarshaler(v) {
		if err := validateCustomMarshaler(v); err != nil {
			return err
		}
	}
	return validateStructFields(v, seen, depth)
}

// isEmbeddedStructField reports whether field is an anonymous struct or
// pointer-to-struct field, which encoding/json flattens rather than skipping
// when the embedded type itself is unexported.
func isEmbeddedStructField(field reflect.StructField) bool {
	if !field.Anonymous {
		return false
	}
	t := field.Type
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct
}

// jsonTagName returns the name portion of a JSON struct tag before the first
// comma.
func jsonTagName(tag string) string {
	if i := strings.IndexByte(tag, ','); i >= 0 {
		return tag[:i]
	}
	return tag
}

// hasJSONOption reports whether a JSON struct tag contains option. The field
// name before the first comma is not an option.
func hasJSONOption(tag, option string) bool {
	i := strings.IndexByte(tag, ',')
	if i < 0 {
		return false
	}
	for opts := tag[i+1:]; opts != ""; {
		var opt string
		if j := strings.IndexByte(opts, ','); j >= 0 {
			opt, opts = opts[:j], opts[j+1:]
		} else {
			opt, opts = opts, ""
		}
		if opt == option {
			return true
		}
	}
	return false
}

// emitsQuotedValue reports whether encoding/json will encode field as a quoted
// JSON string because of the "string" tag option. Only bool, string, integer,
// and float fields support quoting; an unnamed pointer field is followed first.
// A json.Marshaler takes precedence over the plain kind encoder, so those
// values are left to the marshaler validator instead of being skipped.
func emitsQuotedValue(field reflect.StructField, tag string, v reflect.Value) bool {
	if !hasJSONOption(tag, "string") {
		return false
	}
	t := field.Type
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
	default:
		return false
	}
	return !usesJSONMarshaler(v)
}

// usesJSONMarshaler reports whether encoding/json will invoke MarshalJSON on v
// rather than its plain kind encoder. A pointer-receiver MarshalJSON is used
// only when the value is addressable, mirroring encoding/json's conditional
// address encoder.
func usesJSONMarshaler(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	t := v.Type()
	if t.Implements(jsonMarshalerType) {
		return true
	}
	return v.CanAddr() && reflect.PointerTo(t).Implements(jsonMarshalerType)
}

// isEmptyValue mirrors encoding/json's omitempty definition. Unlike
// reflect.Value.IsZero, it does not treat zero structs as empty, because
// encoding/json still emits a zero struct field tagged omitempty.
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	default:
		return false
	}
}

// isZeroer matches the optional IsZero method encoding/json honours for the
// omitzero tag option.
type isZeroer interface {
	IsZero() bool
}

var isZeroerType = reflect.TypeOf((*isZeroer)(nil)).Elem()

// isZeroValue mirrors encoding/json's omitzero check for a field with static
// type t. encoding/json only honours IsZero declared by the field's static
// type, so an interface field does not inherit IsZero from its dynamic value.
// It also boxes a non-addressable value when only the pointer type implements
// IsZero. When no declared IsZero applies, it falls back to reflect's
// structural zero test.
func isZeroValue(t reflect.Type, v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch {
	case t.Kind() == reflect.Interface && t.Implements(isZeroerType):
		// Panic guards mirror encoding/json: a nil interface or an interface
		// holding a nil pointer is zero without calling IsZero.
		if v.IsNil() {
			return true
		}
		if elem := v.Elem(); elem.Kind() == reflect.Pointer && elem.IsNil() {
			return true
		}
		return callIsZero(v)
	case t.Kind() == reflect.Pointer && t.Implements(isZeroerType):
		return v.IsNil() || callIsZero(v)
	case t.Implements(isZeroerType):
		return callIsZero(v)
	case reflect.PointerTo(t).Implements(isZeroerType):
		if !v.CanAddr() {
			// Mirror encoding/json: box a copy so its address can be taken
			// even when the struct field itself is not addressable.
			boxed := reflect.New(v.Type()).Elem()
			boxed.Set(v)
			v = boxed
		}
		return callIsZero(v.Addr())
	}
	return v.IsZero()
}

// callIsZero invokes IsZero when v can be interfaced. Values that cannot be
// interfaced (for example fields reached through an unexported embedded struct)
// fail closed: reporting non-zero makes the caller validate the field rather
// than skip it.
func callIsZero(v reflect.Value) bool {
	if !v.CanInterface() {
		return false
	}
	z, ok := v.Interface().(isZeroer)
	if !ok {
		return false
	}
	return z.IsZero()
}

func validateMapKey(k reflect.Value) error {
	for k.Kind() == reflect.Interface {
		if k.IsNil() {
			return nil
		}
		k = k.Elem()
	}
	// encoding/json encodes a nil pointer map key as the empty string without
	// invoking its TextMarshaler, so mirror that before interfacing or calling
	// the method.
	if k.Kind() == reflect.Pointer && k.IsNil() {
		return nil
	}
	// encoding/json uses the raw string for string-kind keys before consulting
	// any TextMarshaler, so check the underlying string first. A type that
	// implements TextMarshaler but also has string kind is encoded from its
	// raw string value, not from MarshalText.
	if k.Kind() == reflect.String {
		if !utf8.ValidString(k.String()) {
			return &Error{
				Class:   ClassInvalidUTF8,
				Message: "map key is not valid UTF-8",
			}
		}
		return nil
	}
	// encoding/json encodes non-string map keys that implement
	// encoding.TextMarshaler through the text they emit, not through their
	// underlying Go kind. Validate that emitted text so an invalid-UTF-8 key
	// cannot bypass the string check.
	tm, ok, err := textMarshalerFor(k)
	if err != nil {
		return err
	}
	if ok {
		return validateTextMarshaler(tm)
	}
	return nil
}

// textMarshalerFor returns the encoding.TextMarshaler implemented by k, if any.
// When a TextMarshaler applies but cannot be interfaced, it fails closed rather
// than panicking on reflect.Value.Interface.
func textMarshalerFor(k reflect.Value) (encoding.TextMarshaler, bool, error) {
	if !k.IsValid() {
		return nil, false, nil
	}
	t := k.Type()
	if t.Implements(textMarshalerType) {
		if !k.CanInterface() {
			return nil, false, &Error{
				Class:   ClassInvalidJSON,
				Message: fmt.Sprintf("text marshaler %s cannot be interfaced for validation", t),
			}
		}
		return k.Interface().(encoding.TextMarshaler), true, nil
	}
	if k.CanAddr() {
		if pt := reflect.PointerTo(t); pt.Implements(textMarshalerType) {
			if !k.Addr().CanInterface() {
				return nil, false, &Error{
					Class:   ClassInvalidJSON,
					Message: fmt.Sprintf("text marshaler %s cannot be interfaced for validation", pt),
				}
			}
			return k.Addr().Interface().(encoding.TextMarshaler), true, nil
		}
	}
	return nil, false, nil
}

func cyclicValueError() error {
	return &Error{Class: ClassInvalidJSON, Message: "cyclic value"}
}

func depthLimitError() error {
	return &Error{Class: ClassDepthLimit, Message: "maximum nesting depth exceeded"}
}

// validateNumberLiteral enforces the strict numeric policy on a json.Number
// token. The token must be exactly one JSON number with no surrounding
// whitespace; Canonicalize is then used to apply the numeric policy.
func validateNumberLiteral(literal string) error {
	if literal == "" {
		return &Error{Class: ClassInvalidJSON, Message: "empty number literal"}
	}
	if literal[0] != '-' && !isDigit(literal[0]) {
		return &Error{Class: ClassInvalidJSON, Message: "invalid number literal"}
	}
	for i := 0; i < len(literal); i++ {
		switch literal[i] {
		case ' ', '\t', '\n', '\r':
			return &Error{Class: ClassInvalidJSON, Message: "invalid number literal"}
		}
	}
	_, err := Canonicalize([]byte(literal))
	if ErrorClass(err) == ClassTrailingData {
		// A json.Number must be exactly one number token. Trailing junk such as
		// "1x", "1.2.3", or "1]" is malformed in value mode rather than a
		// standalone document followed by trailing data.
		return &Error{Class: ClassInvalidJSON, Message: "invalid number literal"}
	}
	return err
}

// validateRawFragment validates an embedded JSON fragment held in a
// json.RawMessage. Empty or malformed bytes are reported as invalid_json.
func validateRawFragment(raw []byte) error {
	_, err := Canonicalize(raw)
	return err
}

// validateCustomMarshaler validates the dynamic JSON/text produced by a type
// that implements json.Marshaler or encoding.TextMarshaler. Nil pointers are
// never invoked.
//
// A custom marshaler is invoked once here for validation and again by
// encoding/json when CanonicalizeValue marshals the value, so it must be
// deterministic. Its result is deliberately not cached or reused: doing so
// would change encoding/json's public semantics for custom marshalers.
func validateCustomMarshaler(v reflect.Value) error {
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil
	}
	t := v.Type()
	pt := reflect.PointerTo(t)
	valueMarshaler := t.Implements(jsonMarshalerType) || t.Implements(textMarshalerType)
	pointerMarshaler := pt.Implements(jsonMarshalerType) || pt.Implements(textMarshalerType)

	// Prefer the addressable pointer form when it is interfaceable. A value
	// reached through an unexported embedded struct can itself be uninterfaceable
	// while its address is still reachable for validation.
	if v.CanAddr() && v.Addr().CanInterface() {
		switch {
		case pt.Implements(jsonMarshalerType):
			return validateJSONMarshaler(v.Addr().Interface().(json.Marshaler))
		case pt.Implements(textMarshalerType):
			return validateTextMarshaler(v.Addr().Interface().(encoding.TextMarshaler))
		}
	}

	if v.CanInterface() {
		switch {
		case t.Implements(jsonMarshalerType):
			return validateJSONMarshaler(v.Interface().(json.Marshaler))
		case t.Implements(textMarshalerType):
			return validateTextMarshaler(v.Interface().(encoding.TextMarshaler))
		}
	}

	// A custom marshaler applies, but neither the value nor its address can be
	// interfaced to invoke it. Fail closed rather than skipping the value or
	// panicking on reflect.Value.Interface.
	if valueMarshaler || (v.CanAddr() && pointerMarshaler) {
		return &Error{
			Class:   ClassInvalidJSON,
			Message: fmt.Sprintf("custom marshaler %s cannot be interfaced for validation", t),
		}
	}
	return nil
}

func validateJSONMarshaler(m json.Marshaler) error {
	raw, err := m.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = Canonicalize(raw)
	return err
}

func validateTextMarshaler(m encoding.TextMarshaler) error {
	raw, err := m.MarshalText()
	if err != nil {
		return err
	}
	if !utf8.Valid(raw) {
		return &Error{
			Class:   ClassInvalidUTF8,
			Message: "text marshaler returned invalid UTF-8",
		}
	}
	return nil
}

func hasCustomMarshaler(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	t := v.Type()
	if t.Implements(jsonMarshalerType) || t.Implements(textMarshalerType) {
		return true
	}
	if v.CanAddr() {
		pt := reflect.PointerTo(t)
		if pt.Implements(jsonMarshalerType) || pt.Implements(textMarshalerType) {
			return true
		}
	}
	return false
}

// firstInvalidUTF8 returns the offset of the first malformed UTF-8 byte.
func firstInvalidUTF8(b []byte) int {
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		if r == utf8.RuneError && size == 1 {
			return i
		}
		i += size
	}
	return 0
}

type parser struct {
	data      []byte
	pos       int
	valueMode bool
	depth     int
}

func (p *parser) errf(class, message string) error {
	return &Error{Class: class, Offset: p.pos, Message: message}
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *parser) peekByte() byte {
	if p.pos < len(p.data) {
		return p.data[p.pos]
	}
	return 0
}

func (p *parser) hasPrefix(s string) bool {
	return bytes.HasPrefix(p.data[p.pos:], []byte(s))
}

func (p *parser) consumeLiteral(lit string) bool {
	if p.hasPrefix(lit) {
		p.pos += len(lit)
		return true
	}
	return false
}

func (p *parser) parseValue(buf *bytes.Buffer) error {
	p.skipWhitespace()
	if p.pos >= len(p.data) {
		return p.errf(ClassInvalidJSON, "unexpected end of input")
	}
	c := p.data[p.pos]
	switch {
	case c == 'n':
		if p.consumeLiteral("null") {
			buf.WriteString("null")
			return nil
		}
	case c == 't':
		if p.consumeLiteral("true") {
			buf.WriteString("true")
			return nil
		}
	case c == 'f':
		if p.consumeLiteral("false") {
			buf.WriteString("false")
			return nil
		}
	case c == '"':
		s, err := p.parseString()
		if err != nil {
			return err
		}
		writeString(buf, s)
		return nil
	case c == '[':
		return p.parseArray(buf)
	case c == '{':
		return p.parseObject(buf)
	case p.hasPrefix("NaN") && p.literalBoundary(len("NaN")),
		p.hasPrefix("Infinity") && p.literalBoundary(len("Infinity")),
		p.hasPrefix("-Infinity") && p.literalBoundary(len("-Infinity")):
		return p.errf(ClassNonFiniteNumber, "non-finite number literal")
	case c == '-' || (c >= '0' && c <= '9'):
		return p.parseNumber(buf)
	}
	return p.errf(ClassInvalidJSON, "unexpected token")
}

// literalBoundary reports whether the literal of length n starting at the
// current position ends at a valid JSON token boundary. It prevents NaNx,
// Infinityx and similar malformed tokens from being classified as non-finite
// numbers rather than invalid JSON.
func (p *parser) literalBoundary(n int) bool {
	i := p.pos + n
	if i >= len(p.data) {
		return true
	}
	switch p.data[i] {
	case ' ', '\t', '\n', '\r', ',', ']', '}':
		return true
	default:
		return false
	}
}

func (p *parser) enterContainer() error {
	p.depth++
	if p.depth > maxNestingDepth {
		return p.errf(ClassDepthLimit, "maximum nesting depth exceeded")
	}
	return nil
}

func (p *parser) parseArray(buf *bytes.Buffer) error {
	if err := p.enterContainer(); err != nil {
		return err
	}
	defer func() { p.depth-- }()
	p.pos++ // '['
	p.skipWhitespace()
	buf.WriteByte('[')
	if p.peekByte() == ']' {
		p.pos++
		buf.WriteByte(']')
		return nil
	}
	for i := 0; ; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		if err := p.parseValue(buf); err != nil {
			return err
		}
		p.skipWhitespace()
		switch p.peekByte() {
		case ',':
			p.pos++
		case ']':
			p.pos++
			buf.WriteByte(']')
			return nil
		default:
			return p.errf(ClassInvalidJSON, "expected ',' or ']'")
		}
	}
}

type member struct {
	key   string
	value []byte
}

func (p *parser) parseObject(buf *bytes.Buffer) error {
	if err := p.enterContainer(); err != nil {
		return err
	}
	defer func() { p.depth-- }()
	p.pos++ // '{'
	p.skipWhitespace()
	if p.peekByte() == '}' {
		p.pos++
		buf.WriteString("{}")
		return nil
	}

	var members []member
	seen := make(map[string]struct{})

	for {
		p.skipWhitespace()
		if p.peekByte() != '"' {
			return p.errf(ClassInvalidJSON, "expected object member name")
		}
		key, err := p.parseString()
		if err != nil {
			return err
		}

		p.skipWhitespace()
		if p.peekByte() != ':' {
			return p.errf(ClassInvalidJSON, "expected ':' after object member name")
		}
		p.pos++

		var value bytes.Buffer
		if err := p.parseValue(&value); err != nil {
			return err
		}
		// Duplicate detection runs after the member value has been parsed so
		// that value errors (malformed JSON, unrepresentable numbers) take
		// precedence over the duplicate-key rejection, matching Python.
		if _, duplicate := seen[key]; duplicate {
			return p.errf(ClassDuplicateKey, "duplicate object member name")
		}
		seen[key] = struct{}{}
		members = append(members, member{key: key, value: value.Bytes()})

		p.skipWhitespace()
		switch p.peekByte() {
		case ',':
			p.pos++
		case '}':
			p.pos++
			writeObject(buf, members)
			return nil
		default:
			return p.errf(ClassInvalidJSON, "expected ',' or '}'")
		}
	}
}

func writeObject(buf *bytes.Buffer, members []member) {
	sort.SliceStable(members, func(i, j int) bool {
		return utf16Less(members[i].key, members[j].key)
	})
	buf.WriteByte('{')
	for i, m := range members {
		if i > 0 {
			buf.WriteByte(',')
		}
		writeString(buf, m.key)
		buf.WriteByte(':')
		buf.Write(m.value)
	}
	buf.WriteByte('}')
}

func (p *parser) parseString() (string, error) {
	p.pos++ // opening '"'
	var sb strings.Builder
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		switch {
		case c == '"':
			p.pos++
			return sb.String(), nil
		case c == '\\':
			p.pos++
			if p.pos >= len(p.data) {
				return "", p.errf(ClassInvalidJSON, "truncated escape sequence")
			}
			esc := p.data[p.pos]
			p.pos++
			switch esc {
			case '"':
				sb.WriteByte('"')
			case '\\':
				sb.WriteByte('\\')
			case '/':
				sb.WriteByte('/')
			case 'b':
				sb.WriteByte('\b')
			case 'f':
				sb.WriteByte('\f')
			case 'n':
				sb.WriteByte('\n')
			case 'r':
				sb.WriteByte('\r')
			case 't':
				sb.WriteByte('\t')
			case 'u':
				r, err := p.parseUnicodeEscape()
				if err != nil {
					return "", err
				}
				sb.WriteRune(r)
			default:
				return "", p.errf(ClassInvalidJSON, "invalid escape sequence")
			}
		case c < 0x20:
			return "", p.errf(ClassInvalidJSON, "unescaped control character in string")
		default:
			r, size := utf8.DecodeRune(p.data[p.pos:])
			sb.WriteRune(r)
			p.pos += size
		}
	}
	return "", p.errf(ClassInvalidJSON, "unterminated string")
}

// parseUnicodeEscape consumes the four hex digits of a \uXXXX escape (the
// leading "\u" has already been consumed) and, when the escape is a high
// surrogate, requires a following low-surrogate escape.
func (p *parser) parseUnicodeEscape() (rune, error) {
	first, err := p.readHex4()
	if err != nil {
		return 0, err
	}
	if !utf16.IsSurrogate(rune(first)) {
		return rune(first), nil
	}
	if !(p.pos+1 < len(p.data) && p.data[p.pos] == '\\' && p.data[p.pos+1] == 'u') {
		return 0, p.errf(ClassLoneSurrogate, "unpaired UTF-16 surrogate escape")
	}
	p.pos += 2
	second, err := p.readHex4()
	if err != nil {
		return 0, err
	}
	combined := utf16.DecodeRune(rune(first), rune(second))
	if combined == utf8.RuneError {
		return 0, p.errf(ClassLoneSurrogate, "unpaired UTF-16 surrogate escape")
	}
	return combined, nil
}

func (p *parser) readHex4() (uint16, error) {
	if p.pos+4 > len(p.data) {
		return 0, p.errf(ClassInvalidJSON, "truncated \\u escape")
	}
	var v uint16
	for i := 0; i < 4; i++ {
		c := p.data[p.pos+i]
		var d uint16
		switch {
		case c >= '0' && c <= '9':
			d = uint16(c - '0')
		case c >= 'a' && c <= 'f':
			d = uint16(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = uint16(c-'A') + 10
		default:
			return 0, p.errf(ClassInvalidJSON, "invalid \\u escape")
		}
		v = v<<4 | d
	}
	p.pos += 4
	return v, nil
}

func (p *parser) parseNumber(buf *bytes.Buffer) error {
	start := p.pos

	if p.peekByte() == '-' {
		p.pos++
	}

	// Integer part.
	if p.pos >= len(p.data) {
		return p.errf(ClassInvalidJSON, "truncated number")
	}
	switch {
	case p.data[p.pos] == '0':
		p.pos++
		if c := p.peekByte(); c >= '0' && c <= '9' {
			return p.errf(ClassInvalidJSON, "leading zero in number")
		}
	case p.data[p.pos] >= '1' && p.data[p.pos] <= '9':
		p.pos++
		for isDigit(p.peekByte()) {
			p.pos++
		}
	default:
		return p.errf(ClassInvalidJSON, "invalid number")
	}

	plainInteger := true

	// Fraction part.
	if p.peekByte() == '.' {
		plainInteger = false
		p.pos++
		if !isDigit(p.peekByte()) {
			return p.errf(ClassInvalidJSON, "missing fraction digits")
		}
		for isDigit(p.peekByte()) {
			p.pos++
		}
	}

	// Exponent part.
	if c := p.peekByte(); c == 'e' || c == 'E' {
		plainInteger = false
		p.pos++
		if c := p.peekByte(); c == '+' || c == '-' {
			p.pos++
		}
		if !isDigit(p.peekByte()) {
			return p.errf(ClassInvalidJSON, "missing exponent digits")
		}
		for isDigit(p.peekByte()) {
			p.pos++
		}
	}

	token := p.data[start:p.pos]

	if plainInteger {
		if !p.valueMode {
			if err := checkPlainIntegerPolicy(token); err != nil {
				return err
			}
		}
	}

	f, err := strconv.ParseFloat(string(token), 64)
	if err != nil {
		// Underflow to zero is a finite binary64 value; only overflow and
		// genuine parse failures are unrepresentable.
		if !(f == 0 && errors.Is(err, strconv.ErrRange)) {
			return p.errf(ClassNumberUnrepresentable, "number cannot be represented as a finite binary64 value")
		}
	}
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return p.errf(ClassNumberUnrepresentable, "number cannot be represented as a finite binary64 value")
	}
	if math.Abs(f) == 0 {
		// Keep -0 out of the formatter; ES6 serialises both zeros as "0".
		f = 0
	}
	buf.WriteString(formatNumber(f))
	return nil
}

func checkPlainIntegerPolicy(token []byte) error {
	digits := token
	if len(digits) > 0 && digits[0] == '-' {
		digits = digits[1:]
	}
	magnitude, ok := new(big.Int).SetString(string(digits), 10)
	if !ok {
		return &Error{Class: ClassInvalidJSON, Message: "invalid integer literal"}
	}
	if magnitude.Cmp(maxSafeInteger) > 0 {
		return &Error{
			Class:   ClassNumberOutOfPolicy,
			Message: "plain integer literal exceeds 2^53 - 1",
		}
	}
	return nil
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// formatNumber renders a finite float64 using ECMA-262 Number::toString.
func formatNumber(f float64) string {
	if f == 0 {
		return "0"
	}

	negative := math.Signbit(f)
	abs := math.Abs(f)

	// Obtain the shortest round-tripping digits and decimal exponent.
	scientific := strconv.FormatFloat(abs, 'e', -1, 64)
	ePos := strings.IndexByte(scientific, 'e')
	if ePos < 0 {
		// Defensive fallback; 'e' formatting always includes an exponent.
		if negative {
			return "-" + scientific
		}
		return scientific
	}
	mantissa := scientific[:ePos]
	exponent, err := strconv.Atoi(scientific[ePos+1:])
	if err != nil {
		if negative {
			return "-" + scientific
		}
		return scientific
	}

	digits := strings.Replace(mantissa, ".", "", 1)
	k := len(digits)
	// ES6 decimal-point position: value = digits * 10^(n-k).
	n := exponent + 1

	var out string
	switch {
	case k <= n && n <= 21:
		out = digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		out = digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		out = "0." + strings.Repeat("0", -n) + digits
	default:
		if k == 1 {
			out = digits
		} else {
			out = digits[:1] + "." + digits[1:]
		}
		e := n - 1
		sign := "+"
		if e < 0 {
			sign = "-"
			e = -e
		}
		out += "e" + sign + strconv.Itoa(e)
	}

	if negative {
		return "-" + out
	}
	return out
}

// writeString writes a JSON string with the strict profile's escape rules.
func writeString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			if r < 0x20 {
				buf.WriteString(fmt.Sprintf(`\u%04x`, r))
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte('"')
}

// utf16Less compares two strings by their UTF-16 code-unit sequences.
func utf16Less(a, b string) bool {
	ua := utf16.Encode([]rune(a))
	ub := utf16.Encode([]rune(b))
	for i := 0; i < len(ua) && i < len(ub); i++ {
		if ua[i] != ub[i] {
			return ua[i] < ub[i]
		}
	}
	return len(ua) < len(ub)
}
