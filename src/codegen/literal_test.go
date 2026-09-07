package codegen

import "testing"

// TestCFloatLit — правило «.0 перед f»: FormatFloat даёт "2" для 2.0, а "2f"
// в C невалидно; при наличии точки или экспоненты ничего не дописывается.
func TestCFloatLit(t *testing.T) {
	tests := []struct {
		v    float64
		want string
	}{
		{0, "0.0f"},
		{2, "2.0f"},
		{1.5, "1.5f"},
		{3.14, "3.14f"},
		{0.1, "0.1f"},
		{100000, "100000.0f"},
		{1e6, "1e+06f"},
		{1e-5, "1e-05f"},
		{3.4028235e38, "3.4028235e+38f"},
	}
	for _, tc := range tests {
		if got := cFloatLit(tc.v); got != tc.want {
			t.Errorf("cFloatLit(%v) = %q, want %q", tc.v, got, tc.want)
		}
	}
}

func TestZeroLiteral(t *testing.T) {
	tests := map[string]string{"INT": "0", "int": "0", "REAL": "0.0f", "Real": "0.0f"}
	for typ, want := range tests {
		if got := zeroLiteral(typ); got != want {
			t.Errorf("zeroLiteral(%q) = %q, want %q", typ, got, want)
		}
	}
}
