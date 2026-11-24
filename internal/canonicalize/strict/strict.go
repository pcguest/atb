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
)

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
)

// validateValue walks a Go value and enforces the strict profile's numeric
// policy: integer values must fit the safe-integer range and float values must
// be finite. Types with custom JSON or text marshalling are left to
// encoding/json.
func validateValue(v any) error {
	return validateReflect(reflect.ValueOf(v), make(map[uintptr]bool))
}

func validateReflect(v reflect.Value, seen map[uintptr]bool) error {
	if !v.IsValid() {
		return nil
	}
	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		if hasCustomMarshaler(v) {
			return nil
		}
		ptr := v.Pointer()
		if seen[ptr] {
			// Defer cyclic structures to encoding/json, which rejects them.
			return nil
		}
		seen[ptr] = true
		err := validateReflect(v.Elem(), seen)
		delete(seen, ptr)
		return err
	}

	if hasCustomMarshaler(v) {
		return nil
	}

	switch v.Kind() {
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
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := validateReflect(v.Index(i), seen); err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if err := validateReflect(iter.Value(), seen); err != nil {
				return err
			}
		}
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue
			}
			if field.Tag.Get("json") == "-" {
				continue
			}
			if err := validateReflect(v.Field(i), seen); err != nil {
				return err
			}
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
	case p.hasPrefix("NaN"), p.hasPrefix("Infinity"), p.hasPrefix("-Infinity"):
		return p.errf(ClassNonFiniteNumber, "non-finite number literal")
	case c == '-' || (c >= '0' && c <= '9'):
		return p.parseNumber(buf)
	}
	return p.errf(ClassInvalidJSON, "unexpected token")
}

func (p *parser) parseArray(buf *bytes.Buffer) error {
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
		if _, duplicate := seen[key]; duplicate {
			return p.errf(ClassDuplicateKey, "duplicate object member name")
		}
		seen[key] = struct{}{}

		p.skipWhitespace()
		if p.peekByte() != ':' {
			return p.errf(ClassInvalidJSON, "expected ':' after object member name")
		}
		p.pos++

		var value bytes.Buffer
		if err := p.parseValue(&value); err != nil {
			return err
		}
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
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
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
