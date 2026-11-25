// SPDX-License-Identifier: MIT

package strict

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want string
	}{
		{"zero", 0, "0"},
		{"negative zero", math.Copysign(0, -1), "0"},
		{"one", 1, "1"},
		{"one and a half", 1.5, "1.5"},
		{"one tenth", 0.1, "0.1"},
		{"small exponent", 1e-7, "1e-7"},
		{"largest small fixed", 1e-6, "0.000001"},
		{"small fixed", 1e-5, "0.00001"},
		{"below threshold", 1e20, "100000000000000000000"},
		{"threshold", 1e21, "1e+21"},
		{"above threshold", 1e22, "1e+22"},
		{"well above threshold", 1e23, "1e+23"},
		{"max safe integer", 9007199254740991, "9007199254740991"},
		{"longest shortest fraction", 1.2345678901234567, "1.2345678901234567"},
		{"smallest subnormal", 5e-324, "5e-324"},
		{"max float64", 1.7976931348623157e308, "1.7976931348623157e+308"},
		{"negative fraction", -1.5, "-1.5"},
		{"negative threshold", -1e21, "-1e+21"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := formatNumber(tt.in); got != tt.want {
				t.Fatalf("formatNumber(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCanonicalizeStrings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "short escapes and lowercase control escapes",
			input: `"\"\\\b\f\n\r\t\u0001\u001F"`,
			want:  `"\"\\\b\f\n\r\t\u0001\u001f"`,
		},
		{
			name:  "non-ascii stays literal",
			input: `"café ☃"`,
			want:  `"café ☃"`,
		},
		{
			name:  "solidus is not escaped",
			input: `"a\/b"`,
			want:  `"a/b"`,
		},
		{
			name:  "line and paragraph separators stay literal",
			input: "\"a\u2028b\u2029c\"",
			want:  "\"a\u2028b\u2029c\"",
		},
		{
			name:  "surrogate pair decodes to astral rune",
			input: `"\uD83D\uDE00"`,
			want:  "\"\U0001F600\"",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := Canonicalize([]byte(tt.input))
			if err != nil {
				t.Fatalf("Canonicalize(%q): %v", tt.input, err)
			}
			if string(got) != tt.want {
				t.Fatalf("Canonicalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCanonicalizeKeyOrdering(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "code unit ordering",
			input: `{"b":1,"a":2,"C":3,"c":4}`,
			want:  `{"C":3,"a":2,"b":1,"c":4}`,
		},
		{
			name:  "NUL code unit is not a terminator",
			input: `{"a\u0000":1,"a":2}`,
			want:  `{"a":2,"a\u0000":1}`,
		},
		{
			name:  "astral surrogate units sort before U+FFFF",
			input: `{"z":1,"\uFFFF":2,"\uD83D\uDE00":3}`,
			want:  "{\"z\":1,\"\U0001F600\":3,\"\uFFFF\":2}",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := Canonicalize([]byte(tt.input))
			if err != nil {
				t.Fatalf("Canonicalize(%q): %v", tt.input, err)
			}
			if string(got) != tt.want {
				t.Fatalf("Canonicalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCanonicalizeNumbers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"one e twenty", `{"a":1e20}`, `{"a":100000000000000000000}`},
		{"one e twenty one", `{"a":1e21}`, `{"a":1e+21}`},
		{"one e twenty two", `{"a":1e22}`, `{"a":1e+22}`},
		{"negative zero", `{"a":-0}`, `{"a":0}`},
		{"small exponent", `{"a":1e-7}`, `{"a":1e-7}`},
		{"max safe integer", `{"a":9007199254740991}`, `{"a":9007199254740991}`},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := Canonicalize([]byte(tt.input))
			if err != nil {
				t.Fatalf("Canonicalize(%q): %v", tt.input, err)
			}
			if string(got) != tt.want {
				t.Fatalf("Canonicalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCanonicalizeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"duplicate key", []byte(`{"a":1,"a":2}`), ClassDuplicateKey},
		{"duplicate escaped-equivalent key", []byte(`{"a":1,"\u0061":2}`), ClassDuplicateKey},
		{"duplicate nested key", []byte(`{"x":{"k":1,"k":2}}`), ClassDuplicateKey},
		{"invalid utf8", []byte{'{', '"', 's', '"', ':', '"', 0xC3, 0x28, '"', '}'}, ClassInvalidUTF8},
		{"lone high surrogate", []byte(`{"s":"\uD800"}`), ClassLoneSurrogate},
		{"lone low surrogate", []byte(`{"s":"\uDC00"}`), ClassLoneSurrogate},
		{"high surrogate then non-low escape", []byte(`{"s":"\uD800\u0041"}`), ClassLoneSurrogate},
		{"trailing data", []byte(`{"a":1} {"b":2}`), ClassTrailingData},
		{"NaN", []byte(`{"n":NaN}`), ClassNonFiniteNumber},
		{"Infinity", []byte(`{"n":Infinity}`), ClassNonFiniteNumber},
		{"negative Infinity", []byte(`{"n":-Infinity}`), ClassNonFiniteNumber},
		{"2^53", []byte(`{"n":9007199254740992}`), ClassNumberOutOfPolicy},
		{"2^53+1", []byte(`{"n":9007199254740993}`), ClassNumberOutOfPolicy},
		{"negative 2^53", []byte(`{"n":-9007199254740992}`), ClassNumberOutOfPolicy},
		{"overflowing exponent", []byte(`{"n":1e999}`), ClassNumberUnrepresentable},
		{"overflowing plain integer exponent", []byte(`{"n":1E999}`), ClassNumberUnrepresentable},
		{"leading zero", []byte(`{"n":01}`), ClassInvalidJSON},
		{"trailing comma object", []byte(`{"a":1,}`), ClassInvalidJSON},
		{"trailing comma array", []byte(`[1,]`), ClassInvalidJSON},
		{"unquoted key", []byte(`{a:1}`), ClassInvalidJSON},
		{"missing colon", []byte(`{"a" 1}`), ClassInvalidJSON},
		{"empty input", []byte(``), ClassInvalidJSON},
		{"unterminated string", []byte(`"abc`), ClassInvalidJSON},
		{"raw control in string", []byte{'"', 'a', '\n', 'b', '"'}, ClassInvalidJSON},
		{"invalid escape", []byte(`"\q"`), ClassInvalidJSON},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := Canonicalize(tt.input)
			if err == nil {
				t.Fatalf("Canonicalize(%q) succeeded, want error class %q", tt.input, tt.want)
			}
			if got := ErrorClass(err); got != tt.want {
				t.Fatalf("ErrorClass(Canonicalize(%q)) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCanonicalizeValue(t *testing.T) {
	t.Run("finite e20 accepted", func(t *testing.T) {
		got, err := CanonicalizeValue(1e20)
		if err != nil {
			t.Fatalf("CanonicalizeValue(1e20): %v", err)
		}
		if string(got) != "100000000000000000000" {
			t.Fatalf("CanonicalizeValue(1e20) = %q", got)
		}
	})

	t.Run("NaN rejected", func(t *testing.T) {
		_, err := CanonicalizeValue(math.NaN())
		if got := ErrorClass(err); got != ClassNonFiniteNumber {
			t.Fatalf("ErrorClass = %q, want %q", got, ClassNonFiniteNumber)
		}
	})

	t.Run("Infinity rejected", func(t *testing.T) {
		_, err := CanonicalizeValue(math.Inf(1))
		if got := ErrorClass(err); got != ClassNonFiniteNumber {
			t.Fatalf("ErrorClass = %q, want %q", got, ClassNonFiniteNumber)
		}
	})

	t.Run("integer outside policy rejected", func(t *testing.T) {
		_, err := CanonicalizeValue(int64(9007199254740992))
		if got := ErrorClass(err); got != ClassNumberOutOfPolicy {
			t.Fatalf("ErrorClass = %q, want %q", got, ClassNumberOutOfPolicy)
		}
	})
}

func TestCanonicalizeValueJSONNumberPolicy(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		want      string
		wantClass string
	}{
		{"out of policy", json.Number("9007199254740993"), "", ClassNumberOutOfPolicy},
		{"max safe integer", json.Number("9007199254740991"), "9007199254740991", ""},
		{"finite exponent", json.Number("1e20"), "100000000000000000000", ""},
		{"leading whitespace", json.Number(" 1"), "", ClassInvalidJSON},
		{"trailing whitespace", json.Number("1 "), "", ClassInvalidJSON},
		{"inner whitespace", json.Number("1 2"), "", ClassInvalidJSON},
		{"trailing letter", json.Number("1x"), "", ClassInvalidJSON},
		{"multiple decimal points", json.Number("1.2.3"), "", ClassInvalidJSON},
		{"trailing bracket", json.Number("1]"), "", ClassInvalidJSON},
		{"null literal", json.Number("null"), "", ClassInvalidJSON},
		{"true literal", json.Number("true"), "", ClassInvalidJSON},
		{"array literal", json.Number("[]"), "", ClassInvalidJSON},
		{"object literal", json.Number("{}"), "", ClassInvalidJSON},
		{"empty literal", json.Number(""), "", ClassInvalidJSON},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := CanonicalizeValue(tt.value)
			if tt.wantClass != "" {
				if class := ErrorClass(err); class != tt.wantClass {
					t.Fatalf("ErrorClass(CanonicalizeValue(%v)) = %q, want %q (%v)", tt.value, class, tt.wantClass, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("CanonicalizeValue(%v): %v", tt.value, err)
			}
			if string(got) != tt.want {
				t.Fatalf("CanonicalizeValue(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestCanonicalizeValueNestedJSONNumberPolicy(t *testing.T) {
	type payload struct {
		N json.Number `json:"n"`
	}
	_, err := CanonicalizeValue(payload{N: json.Number("9007199254740993")})
	if got := ErrorClass(err); got != ClassNumberOutOfPolicy {
		t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassNumberOutOfPolicy, err)
	}

	_, err = CanonicalizeValue(map[string]any{"n": json.Number("9007199254740993")})
	if got := ErrorClass(err); got != ClassNumberOutOfPolicy {
		t.Fatalf("ErrorClass(map) = %q, want %q (%v)", got, ClassNumberOutOfPolicy, err)
	}
}

func TestCanonicalizeValueEmbeddedUnexportedStruct(t *testing.T) {
	type hidden struct {
		N int64  `json:"n"`
		S string `json:"s"`
	}
	type doc struct {
		hidden
	}

	t.Run("numeric policy applies", func(t *testing.T) {
		_, err := CanonicalizeValue(doc{hidden{N: 9007199254740993}})
		if got := ErrorClass(err); got != ClassNumberOutOfPolicy {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassNumberOutOfPolicy, err)
		}
	})

	t.Run("invalid utf8 in exported field rejected", func(t *testing.T) {
		_, err := CanonicalizeValue(doc{hidden{S: "a\xffb"}})
		if got := ErrorClass(err); got != ClassInvalidUTF8 {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassInvalidUTF8, err)
		}
	})

	t.Run("valid embedded value canonicalised", func(t *testing.T) {
		got, err := CanonicalizeValue(doc{hidden{N: 42, S: "ok"}})
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `{"n":42,"s":"ok"}`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})
}

func TestCanonicalizeValueRawMessagePolicy(t *testing.T) {
	t.Run("out of policy", func(t *testing.T) {
		_, err := CanonicalizeValue(json.RawMessage(`{"n":9007199254740993}`))
		if got := ErrorClass(err); got != ClassNumberOutOfPolicy {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassNumberOutOfPolicy, err)
		}
	})

	t.Run("empty raw message", func(t *testing.T) {
		_, err := CanonicalizeValue(json.RawMessage(``))
		if got := ErrorClass(err); got != ClassInvalidJSON {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassInvalidJSON, err)
		}
	})

	t.Run("nil raw message is null", func(t *testing.T) {
		got, err := CanonicalizeValue(json.RawMessage(nil))
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := "null"; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})

	t.Run("nil raw message field is null", func(t *testing.T) {
		type payload struct {
			R json.RawMessage `json:"r"`
		}
		got, err := CanonicalizeValue(payload{})
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `{"r":null}`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})

	t.Run("empty non-nil raw message remains invalid", func(t *testing.T) {
		_, err := CanonicalizeValue(json.RawMessage{})
		if got := ErrorClass(err); got != ClassInvalidJSON {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassInvalidJSON, err)
		}
	})

	t.Run("valid raw message canonicalised", func(t *testing.T) {
		got, err := CanonicalizeValue(json.RawMessage(`{"b":1,"a":2}`))
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `{"a":2,"b":1}`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})
}

func TestCanonicalizeValueOmitEmpty(t *testing.T) {
	type withOmitEmpty struct {
		N json.Number `json:"n,omitempty"`
	}

	t.Run("zero json number omitted", func(t *testing.T) {
		got, err := CanonicalizeValue(withOmitEmpty{})
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `{}`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})

	t.Run("non-zero json number included", func(t *testing.T) {
		got, err := CanonicalizeValue(withOmitEmpty{N: json.Number("1")})
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `{"n":1}`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})

	t.Run("zero json number without omitempty rejected", func(t *testing.T) {
		type withNumber struct {
			N json.Number `json:"n"`
		}
		_, err := CanonicalizeValue(withNumber{})
		if got := ErrorClass(err); got != ClassInvalidJSON {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassInvalidJSON, err)
		}
	})

	t.Run("zero json number omitted with omitzero", func(t *testing.T) {
		type withOmitZero struct {
			N json.Number `json:"n,omitzero"`
		}
		got, err := CanonicalizeValue(withOmitZero{})
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `{}`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})
}

func TestCanonicalizeValueTagOptionParsing(t *testing.T) {
	t.Run("field named omitempty is not an option", func(t *testing.T) {
		type payload struct {
			OmitEmpty json.Number `json:"omitempty"`
		}
		_, err := CanonicalizeValue(payload{})
		if got := ErrorClass(err); got != ClassInvalidJSON {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassInvalidJSON, err)
		}
	})

	t.Run("field named omitzero is not an option", func(t *testing.T) {
		type payload struct {
			OmitZero json.Number `json:"omitzero"`
		}
		_, err := CanonicalizeValue(payload{})
		if got := ErrorClass(err); got != ClassInvalidJSON {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassInvalidJSON, err)
		}
	})
}

func TestCanonicalizeValueOmitZeroCustomIsZero(t *testing.T) {
	const large = int64(9007199254740993)

	t.Run("IsZero true omits field containing a large integer", func(t *testing.T) {
		type payload struct {
			Z alwaysZeroLargeInt `json:"z,omitzero"`
		}
		value := payload{Z: alwaysZeroLargeInt{N: large}}

		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("encoding/json rejects the fixture: %v", err)
		}
		if want := `{}`; string(raw) != want {
			t.Fatalf("encoding/json = %s, want %s", raw, want)
		}

		got, err := CanonicalizeValue(value)
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `{}`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})

	t.Run("IsZero false does not bypass marshaler policy", func(t *testing.T) {
		type payload struct {
			Z neverZeroLargeJSON `json:"z,omitzero"`
		}
		value := payload{Z: neverZeroLargeJSON{N: large}}

		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("encoding/json rejects the fixture: %v", err)
		}
		if want := `{"z":9007199254740993}`; string(raw) != want {
			t.Fatalf("encoding/json = %s, want %s", raw, want)
		}

		_, err = CanonicalizeValue(value)
		if got := ErrorClass(err); got != ClassNumberOutOfPolicy {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassNumberOutOfPolicy, err)
		}
	})
}

func TestCanonicalizeValueOmitZeroStaticType(t *testing.T) {
	const large = int64(9007199254740993)

	t.Run("interface field does not inherit dynamic IsZero", func(t *testing.T) {
		type payload struct {
			V any `json:"v,omitzero"`
		}
		value := payload{V: dynamicIsZero{N: large}}

		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("encoding/json rejects the fixture: %v", err)
		}
		if want := `{"v":{"n":9007199254740993}}`; string(raw) != want {
			t.Fatalf("encoding/json = %s, want %s", raw, want)
		}

		_, err = CanonicalizeValue(value)
		if got := ErrorClass(err); got != ClassNumberOutOfPolicy {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassNumberOutOfPolicy, err)
		}
	})

	t.Run("pointer receiver IsZero on non-addressable field", func(t *testing.T) {
		type payload struct {
			Z pointerReceiverIsZero `json:"z,omitzero"`
		}
		value := payload{Z: pointerReceiverIsZero{N: large}}

		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("encoding/json rejects the fixture: %v", err)
		}
		if want := `{}`; string(raw) != want {
			t.Fatalf("encoding/json = %s, want %s", raw, want)
		}

		got, err := CanonicalizeValue(value)
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `{}`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})
}

func TestCanonicalizeValueStringOptionNumeric(t *testing.T) {
	const large = int64(9007199254740993)
	type payload struct {
		N int64 `json:"n,string"`
	}
	value := payload{N: large}

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encoding/json rejects the fixture: %v", err)
	}
	if want := `{"n":"9007199254740993"}`; string(raw) != want {
		t.Fatalf("encoding/json = %s, want %s", raw, want)
	}

	got, err := CanonicalizeValue(value)
	if err != nil {
		t.Fatalf("CanonicalizeValue: %v", err)
	}
	if want := `{"n":"9007199254740993"}`; string(got) != want {
		t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
	}
}

func TestCanonicalizeValueStringKindMapKeyPrecedence(t *testing.T) {
	t.Run("string-kind key does not reject via TextMarshaler", func(t *testing.T) {
		value := map[shadowedTextMarshalerString]int{"ok": 1}
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("encoding/json rejects the fixture: %v", err)
		}
		got, err := CanonicalizeValue(value)
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		// The raw string key is used before MarshalText for string-kind keys.
		// Some encoding/json implementations still consult MarshalText here, so
		// assert agreement with encoding/json rather than a fixed spelling.
		if string(got) != string(raw) {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, raw)
		}
	})
}

func TestCanonicalizeValueInvalidUTF8(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		_, err := CanonicalizeValue("a\xffb")
		if got := ErrorClass(err); got != ClassInvalidUTF8 {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassInvalidUTF8, err)
		}
	})

	t.Run("string map key", func(t *testing.T) {
		_, err := CanonicalizeValue(map[string]int{"a\xffb": 1})
		if got := ErrorClass(err); got != ClassInvalidUTF8 {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassInvalidUTF8, err)
		}
	})

	t.Run("text marshaler map key emits invalid utf8", func(t *testing.T) {
		_, err := CanonicalizeValue(map[textMarshalerString]int{"a\xffb": 1})
		if got := ErrorClass(err); got != ClassInvalidUTF8 {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassInvalidUTF8, err)
		}
	})

	t.Run("text marshaler map key accepted when valid", func(t *testing.T) {
		got, err := CanonicalizeValue(map[textMarshalerString]int{"b": 1, "a": 2})
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `{"a":2,"b":1}`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})
}

func TestCanonicalizeValueTextMarshalerMapKeyPointer(t *testing.T) {
	t.Run("nil pointer key is empty and accepted", func(t *testing.T) {
		got, err := CanonicalizeValue(map[*textMarshalerKey]int{nil: 1})
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `{"":1}`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})

	t.Run("non-nil pointer key uses MarshalText", func(t *testing.T) {
		a, b := textMarshalerKey("a"), textMarshalerKey("b")
		got, err := CanonicalizeValue(map[*textMarshalerKey]int{&a: 1, &b: 2})
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `{"a":1,"b":2}`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})
}

func TestCanonicalizeValueCycleSafety(t *testing.T) {
	t.Run("map cycle", func(t *testing.T) {
		m := map[string]any{}
		m["self"] = m
		_, err := CanonicalizeValue(m)
		if got := ErrorClass(err); got != ClassInvalidJSON {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassInvalidJSON, err)
		}
	})

	t.Run("slice cycle", func(t *testing.T) {
		a := make([]any, 1)
		a[0] = a
		_, err := CanonicalizeValue(a)
		if got := ErrorClass(err); got != ClassInvalidJSON {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassInvalidJSON, err)
		}
	})

	t.Run("pointer cycle", func(t *testing.T) {
		type node struct {
			Next *node `json:"next"`
		}
		n := &node{}
		n.Next = n
		_, err := CanonicalizeValue(n)
		if got := ErrorClass(err); got != ClassInvalidJSON {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassInvalidJSON, err)
		}
	})

	t.Run("pointer to first field is not a cycle", func(t *testing.T) {
		type inner struct {
			X int `json:"X"`
		}
		type outer struct {
			In inner  `json:"In"`
			P  *inner `json:"P"`
		}
		o := &outer{}
		o.P = &o.In
		got, err := CanonicalizeValue(o)
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `{"In":{"X":0},"P":{"X":0}}`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})

	t.Run("shared non-cyclic value accepted", func(t *testing.T) {
		shared := map[string]int{"a": 1}
		got, err := CanonicalizeValue([]any{shared, shared})
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `[{"a":1},{"a":1}]`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})
}

func TestCanonicalizeDepthLimit(t *testing.T) {
	raw := func(depth int) []byte {
		return []byte(strings.Repeat("[", depth) + strings.Repeat("]", depth))
	}

	t.Run("raw 512 accepted", func(t *testing.T) {
		if _, err := Canonicalize(raw(maxNestingDepth)); err != nil {
			t.Fatalf("Canonicalize(%d nested arrays): %v", maxNestingDepth, err)
		}
	})

	t.Run("raw 513 rejected", func(t *testing.T) {
		_, err := Canonicalize(raw(maxNestingDepth + 1))
		if got := ErrorClass(err); got != ClassDepthLimit {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassDepthLimit, err)
		}
	})

	t.Run("very deep raw arrays rejected without crashing", func(t *testing.T) {
		_, err := Canonicalize(raw(10 * maxNestingDepth))
		if got := ErrorClass(err); got != ClassDepthLimit {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassDepthLimit, err)
		}
	})

	t.Run("value 512 accepted", func(t *testing.T) {
		if _, err := CanonicalizeValue(nestedArrays(maxNestingDepth)); err != nil {
			t.Fatalf("CanonicalizeValue(%d nested arrays): %v", maxNestingDepth, err)
		}
	})

	t.Run("value 513 rejected", func(t *testing.T) {
		_, err := CanonicalizeValue(nestedArrays(maxNestingDepth + 1))
		if got := ErrorClass(err); got != ClassDepthLimit {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassDepthLimit, err)
		}
	})
}

func nestedArrays(depth int) any {
	if depth <= 0 {
		return []any{}
	}
	var v any = []any{}
	for i := 1; i < depth; i++ {
		v = []any{v}
	}
	return v
}

// makePointerChain builds a linear chain of depth non-nil *any pointers ending
// in a nil interface, for depth-limit coverage of pointer traversal.
func makePointerChain(depth int) any {
	var v any
	for i := 0; i < depth; i++ {
		next := v
		v = &next
	}
	return v
}

func TestCanonicalizeValuePointerDepthLimit(t *testing.T) {
	t.Run("512-deep pointer chain accepted", func(t *testing.T) {
		got, err := CanonicalizeValue(makePointerChain(maxNestingDepth))
		if err != nil {
			t.Fatalf("CanonicalizeValue(%d-deep pointer chain): %v", maxNestingDepth, err)
		}
		if want := "null"; string(got) != want {
			t.Fatalf("CanonicalizeValue(%d-deep pointer chain) = %q, want %q", maxNestingDepth, got, want)
		}
	})

	t.Run("513-deep pointer chain rejected", func(t *testing.T) {
		_, err := CanonicalizeValue(makePointerChain(maxNestingDepth + 1))
		if got := ErrorClass(err); got != ClassDepthLimit {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassDepthLimit, err)
		}
	})
}

func TestCanonicalizeDuplicatePrecedence(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"duplicate after unrepresentable value", `{"a":1,"a":1e999}`, ClassNumberUnrepresentable},
		{"duplicate after malformed value", `{"a":1,"a":}`, ClassInvalidJSON},
		{"plain duplicate", `{"a":1,"a":2}`, ClassDuplicateKey},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := Canonicalize([]byte(tt.input))
			if got := ErrorClass(err); got != tt.want {
				t.Fatalf("ErrorClass(Canonicalize(%q)) = %q, want %q (%v)", tt.input, got, tt.want, err)
			}
		})
	}
}

func TestCanonicalizeNonFiniteBoundaries(t *testing.T) {
	t.Run("exact literals are non-finite", func(t *testing.T) {
		for _, input := range []string{`NaN`, `Infinity`, `-Infinity`, `[NaN]`, `{"n":Infinity}`} {
			_, err := Canonicalize([]byte(input))
			if got := ErrorClass(err); got != ClassNonFiniteNumber {
				t.Fatalf("ErrorClass(Canonicalize(%q)) = %q, want %q", input, got, ClassNonFiniteNumber)
			}
		}
	})

	t.Run("bad token boundary is invalid json", func(t *testing.T) {
		for _, input := range []string{`NaNx`, `Infinityx`, `-Infinityx`, `NaN1`} {
			_, err := Canonicalize([]byte(input))
			if got := ErrorClass(err); got != ClassInvalidJSON {
				t.Fatalf("ErrorClass(Canonicalize(%q)) = %q, want %q", input, got, ClassInvalidJSON)
			}
		}
	})
}

func TestParseFloatUnderflow(t *testing.T) {
	// Go's strconv.ParseFloat currently reports underflow as a silent zero
	// (nil error) rather than ErrRange. Both outcomes are finite and must be
	// accepted as 0 by the strict parser.
	f, err := strconv.ParseFloat("1e-999", 64)
	if f != 0 {
		t.Fatalf("ParseFloat(%q) = %v; want 0", "1e-999", f)
	}
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		t.Fatalf("ParseFloat(%q) error = %v; want nil or ErrRange", "1e-999", err)
	}
	t.Logf("ParseFloat(\"1e-999\") = %v, %v", f, err)
}

func TestCanonicalizeUnderflow(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input string
		want  string
	}{
		{"positive underflow", `{"n":1e-999}`, `{"n":0}`},
		{"negative underflow", `{"n":-1e-999}`, `{"n":0}`},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := Canonicalize([]byte(tt.input))
			if err != nil {
				t.Fatalf("Canonicalize(%q): %v", tt.input, err)
			}
			if string(got) != tt.want {
				t.Fatalf("Canonicalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCanonicalizeValueCustomMarshalerPolicy(t *testing.T) {
	t.Run("json marshaler fragment is validated", func(t *testing.T) {
		_, err := CanonicalizeValue(jsonMarshalerFunc(func() ([]byte, error) {
			return []byte(`9007199254740993`), nil
		}))
		if got := ErrorClass(err); got != ClassNumberOutOfPolicy {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassNumberOutOfPolicy, err)
		}
	})

	t.Run("text marshaler invalid utf8 rejected", func(t *testing.T) {
		_, err := CanonicalizeValue(textMarshalerString("a\xffb"))
		if got := ErrorClass(err); got != ClassInvalidUTF8 {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassInvalidUTF8, err)
		}
	})
}

type jsonMarshalerFunc func() ([]byte, error)

func (f jsonMarshalerFunc) MarshalJSON() ([]byte, error) { return f() }

type textMarshalerString string

func (s textMarshalerString) MarshalText() ([]byte, error) { return []byte(s), nil }

// shadowedTextMarshalerString is a string-kind map key whose MarshalText would
// emit invalid UTF-8. encoding/json ignores MarshalText for string-kind keys,
// so the raw string key is encoded and must be accepted.
type shadowedTextMarshalerString string

func (shadowedTextMarshalerString) MarshalText() ([]byte, error) {
	return []byte{0xff}, nil
}

// dynamicIsZero implements IsZero, but an interface-typed field must not inherit
// it from the dynamic value: encoding/json only honours IsZero declared by the
// field's static type.
type dynamicIsZero struct {
	N int64 `json:"n"`
}

func (dynamicIsZero) IsZero() bool { return true }

// pointerReceiverIsZero reports zero only through its pointer type, so a
// non-addressable field value must be boxed before IsZero is called.
type pointerReceiverIsZero struct {
	N int64 `json:"n"`
}

func (*pointerReceiverIsZero) IsZero() bool { return true }

// textMarshalerKey implements encoding.TextMarshaler on its pointer type so it
// can be used as a map key, including a nil pointer key.
type textMarshalerKey string

func (k *textMarshalerKey) MarshalText() ([]byte, error) {
	if k == nil {
		return nil, nil
	}
	return []byte(*k), nil
}

// alwaysZeroLargeInt reports itself as zero regardless of its fields, so
// omitzero must trust IsZero and omit it instead of validating the large field.
type alwaysZeroLargeInt struct {
	N int64 `json:"n"`
}

func (alwaysZeroLargeInt) IsZero() bool { return true }

// neverZeroLargeJSON reports itself as non-zero and emits a plain integer
// outside the safe-integer policy through its custom marshaler.
type neverZeroLargeJSON struct {
	N int64 `json:"n"`
}

func (neverZeroLargeJSON) IsZero() bool { return false }

func (neverZeroLargeJSON) MarshalJSON() ([]byte, error) {
	return []byte("9007199254740993"), nil
}

// EmbeddedLargeA and EmbeddedLargeB both implement json.Marshaler at the same
// embedding depth, so a parent that embeds both is not itself a json.Marshaler
// and encoding/json flattens their fields instead of calling the marshalers.
type EmbeddedLargeA struct {
	N int64 `json:"a"`
}

func (EmbeddedLargeA) MarshalJSON() ([]byte, error) { return []byte("0"), nil }

type EmbeddedLargeB struct {
	N int64 `json:"b"`
}

func (EmbeddedLargeB) MarshalJSON() ([]byte, error) { return []byte("0"), nil }

// jsonOnlyMarshaler is a JSON marshaler that deliberately does not implement
// encoding.TextMarshaler, so a promoted method cannot be validated through the
// text path by accident.
type jsonOnlyMarshaler struct{ V string }

func (m jsonOnlyMarshaler) MarshalJSON() ([]byte, error) { return json.Marshal(m.V) }

// TestCanonicalizeValueEmbeddedUnexportedMarshaler guards against a reflection
// panic in validateCustomMarshaler. The value walk can reach a custom marshaler
// through an unexported embedded struct, and that value is not interfaceable
// even though encoding/json accepts the outer value. Validation must fail
// closed with a classified error rather than calling reflect.Value.Interface.
func TestCanonicalizeValueEmbeddedUnexportedMarshaler(t *testing.T) {
	t.Run("uninterfaceable embedded json marshaler fails closed", func(t *testing.T) {
		type wrapped struct{ jsonOnlyMarshaler }
		// The exported field shadows the promoted MarshalJSON method, so the
		// outer type does not implement json.Marshaler and the walk descends
		// into the unexported embedded struct. encoding/json still accepts the
		// value and encodes its fields.
		type doc struct {
			wrapped
			MarshalJSON int `json:"marshal_json"`
		}

		value := doc{wrapped{jsonOnlyMarshaler{V: "x"}}, 0}
		if _, err := json.Marshal(value); err != nil {
			t.Fatalf("encoding/json rejects the fixture: %v", err)
		}

		var (
			got []byte
			err error
		)
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("CanonicalizeValue panicked: %v", r)
				}
			}()
			got, err = CanonicalizeValue(value)
		}()
		if got != nil {
			t.Fatalf("CanonicalizeValue = %q, want no output", got)
		}
		if class := ErrorClass(err); class != ClassInvalidJSON {
			t.Fatalf("ErrorClass = %q, want %q (%v)", class, ClassInvalidJSON, err)
		}
	})

	t.Run("uninterfaceable embedded time value fails closed", func(t *testing.T) {
		// Two unexported embedded structs promote time.Time's MarshalJSON at
		// the same depth, so the selector is ambiguous and the outer type does
		// not implement json.Marshaler. encoding/json still accepts the value.
		type embeddedTimeA struct{ time.Time }
		type embeddedTimeB struct{ time.Time }
		type doc struct {
			embeddedTimeA
			embeddedTimeB
		}

		value := doc{
			embeddedTimeA{time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)},
			embeddedTimeB{time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)},
		}
		//lint:ignore SA9005 intentional encoding/json traversal fixture: embedded time selectors are ambiguous
		if _, err := json.Marshal(value); err != nil {
			t.Fatalf("encoding/json rejects the fixture: %v", err)
		}

		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("CanonicalizeValue panicked: %v", r)
				}
			}()
			_, err = CanonicalizeValue(value)
		}()
		if class := ErrorClass(err); class != ClassInvalidJSON {
			t.Fatalf("ErrorClass = %q, want %q (%v)", class, ClassInvalidJSON, err)
		}
	})

	t.Run("ordinary embedded unexported struct with exported time still works", func(t *testing.T) {
		type hidden struct {
			T time.Time `json:"t"`
		}
		type doc struct {
			hidden
		}

		value := doc{hidden{T: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)}}
		got, err := CanonicalizeValue(value)
		if err != nil {
			t.Fatalf("CanonicalizeValue: %v", err)
		}
		if want := `{"t":"2024-01-02T03:04:05Z"}`; string(got) != want {
			t.Fatalf("CanonicalizeValue = %q, want %q", got, want)
		}
	})
}

func TestCanonicalizeValueFlattenedEmbeddedMarshaler(t *testing.T) {
	const large = int64(9007199254740993)

	t.Run("ambiguous promoted marshalers still validate flattened fields", func(t *testing.T) {
		type doc struct {
			EmbeddedLargeA
			EmbeddedLargeB
		}
		value := doc{
			EmbeddedLargeA{N: large},
			EmbeddedLargeB{N: large},
		}

		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("encoding/json rejects the fixture: %v", err)
		}
		if want := `{"a":9007199254740993,"b":9007199254740993}`; string(raw) != want {
			t.Fatalf("encoding/json = %s, want %s", raw, want)
		}

		_, err = CanonicalizeValue(value)
		if got := ErrorClass(err); got != ClassNumberOutOfPolicy {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassNumberOutOfPolicy, err)
		}
	})

	t.Run("shadowed promoted marshaler still validates flattened fields", func(t *testing.T) {
		type doc struct {
			EmbeddedLargeA
			MarshalJSON int `json:"marshal_json"`
		}
		value := doc{EmbeddedLargeA{N: large}, 0}

		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("encoding/json rejects the fixture: %v", err)
		}
		if want := `{"a":9007199254740993,"marshal_json":0}`; string(raw) != want {
			t.Fatalf("encoding/json = %s, want %s", raw, want)
		}

		_, err = CanonicalizeValue(value)
		if got := ErrorClass(err); got != ClassNumberOutOfPolicy {
			t.Fatalf("ErrorClass = %q, want %q (%v)", got, ClassNumberOutOfPolicy, err)
		}
	})
}

func TestErrorClassUnknown(t *testing.T) {
	if got := ErrorClass(nil); got != "" {
		t.Fatalf("ErrorClass(nil) = %q, want empty", got)
	}
	if got := ErrorClass(errSentinel); got != "" {
		t.Fatalf("ErrorClass(foreign error) = %q, want empty", got)
	}
}

var errSentinel = &sentinelError{}

type sentinelError struct{}

func (*sentinelError) Error() string { return "sentinel" }

// FuzzCanonicalizeNoPanic checks the strict entry point's robustness contract:
// Canonicalize never panics, every rejection carries a stable error class, and
// every accepted result is well-formed UTF-8 JSON. It deliberately does not
// assert that output is re-accepted by Canonicalize: the safe-integer policy
// applies to raw integer tokens only, so an accepted 1e20 canonicalises to a
// plain integer token that the parser would reject on a second pass. That is a
// known policy limitation, not a robustness failure.
func FuzzCanonicalizeNoPanic(f *testing.F) {
	seeds := [][]byte{
		// Representative shared strict vectors.
		[]byte(`{"b":1,"a":[true,false,null],"C":"café ☃"}`),
		[]byte(`{"a":1e20}`),
		[]byte(`{"a":-0}`),
		[]byte(`{"s":"\uD83D\uDE00"}`),
		// Invalid UTF-8 bytes (0xC3 0x28 from the negative vectors).
		[]byte{'{', '"', 's', '"', ':', '"', 0xC3, 0x28, '"', '}'},
		// Rejections: duplicate key, lone surrogate escape, trailing data,
		// bad non-finite token boundary, and a plain integer outside policy.
		[]byte(`{"a":1,"a":2}`),
		[]byte(`{"s":"\uD800"}`),
		[]byte(`{"a":1} {"b":2}`),
		[]byte(`NaNx`),
		[]byte(`{"n":9007199254740992}`),
		// Depth bound boundary: 513 nested arrays must be rejected, not crash.
		[]byte(strings.Repeat("[", maxNestingDepth+1) + strings.Repeat("]", maxNestingDepth+1)),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw []byte) {
		out, err := Canonicalize(raw)
		if err != nil {
			if class := ErrorClass(err); class == "" {
				t.Fatalf("Canonicalize(%q) returned an unclassified error: %v", raw, err)
			}
			return
		}
		if !utf8.Valid(out) {
			t.Fatalf("Canonicalize(%q) returned invalid UTF-8: %q", raw, out)
		}
		if !json.Valid(out) {
			t.Fatalf("Canonicalize(%q) returned invalid JSON: %q", raw, out)
		}
	})
}
