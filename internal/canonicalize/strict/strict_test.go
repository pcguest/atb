// SPDX-License-Identifier: MIT

package strict

import (
	"math"
	"testing"
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
