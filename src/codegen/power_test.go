package codegen_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the generated operations under UBSan, including failing ST programs.
func runPower(t *testing.T, src, stdout, failure string) {
	t.Helper()
	gcc, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc not found")
	}
	code := generateSource(t, src, 1)
	dir := t.TempDir()
	source, bin := filepath.Join(dir, "power.c"), filepath.Join(dir, "power")
	if err := os.WriteFile(source, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(gcc, "-std=c99", "-Wall", "-Wextra", "-Werror", "-fsanitize=undefined,float-cast-overflow", "-fno-sanitize-recover=all", source, "-lm", "-o", bin).CombinedOutput(); err != nil {
		t.Fatalf("gcc: %v\n%s", err, out)
	}
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	wantStatus := 0
	if failure != "" {
		wantStatus = 1
	}
	if ctx.Err() != nil || cmd.ProcessState == nil || cmd.ProcessState.ExitCode() != wantStatus || string(out) != stdout || stderr.String() != failure {
		t.Fatalf("run: %v, stdout %q, stderr %q", err, out, stderr.String())
	}
}

func TestPowerBoundaries(t *testing.T) {
	runPower(t, `FUNCTION Identity : REAL VAR_INPUT x : REAL; END_VAR Identity := x; END_FUNCTION
PROGRAM P VAR ok : BOOL; i : INT; END_VAR
ok := 0 ** 3 = 0 AND (-1) ** -3 = -1 AND (-1) ** -2 = 1 AND 1 ** -32768 = 1
 AND (-1) ** 1e30 = 1 AND (-1) ** 32767 = -1
 AND (-2) ** 3.0 = -8 AND (-2) ** -2 = 0.25 AND 2 ** -3 = 0.125
 AND 2 ** (3 ** 2) = 512 AND 2 ** (1/2) = 1
 AND 2 ** (1.0/2) > 1.4 AND 2 ** (1.0/2) < 1.42
 AND Identity(2 ** 127) = 1.7014118e38 AND 2 ** -149 > 0 AND 2 ** -150 = 0
 AND (+5 MOD 3) ** 2 = 4;
WHILE 2 ** i < 8 DO i := i + 1; END_WHILE
IF FALSE THEN ok := (0 ** -1) = 1; END_IF
IF FALSE AND 0 ** -1 > 1 THEN ok := FALSE; END_IF
IF TRUE OR (-2) ** 0.5 > 0 THEN ELSE ok := FALSE; END_IF
END_PROGRAM`, "ok=1\ni=3\n", "")
}

func TestPowerRuntimeErrors(t *testing.T) {
	const domain = "**: domain error\n"
	const overflow = "**: range error (overflow)\n"
	const nonfinite = "**: range error (non-finite operand)\n"
	for _, tc := range []struct{ expr, want string }{
		{"0 ** -1", domain}, {"(-2) ** 0.5", domain}, {"2 ** 128", overflow},
		{"(x-x) ** (1-x)", domain}, {"(-x) ** (1.0/x)", domain}, {"x ** (64*x)", overflow},
		{"(big * big) ** 0", nonfinite}, {"1 ** (big * big)", nonfinite},
		// Both operands fail. The base must fail first.
		{"(x ** 128) ** (0 ** -1)", overflow},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			runPower(t, "PROGRAM P VAR x : REAL := 2; big : REAL := 1e30; r : REAL; END_VAR r := "+tc.expr+"; END_PROGRAM", "", tc.want)
		})
	}
}

func TestPowerInitializerNames(t *testing.T) {
	runPower(t, `FUNCTION powf : REAL VAR_INPUT st_powf : REAL; END_VAR
VAR truncf : REAL := st_powf ** 2; END_VAR powf := truncf;
END_FUNCTION
PROGRAM st2c_pow
VAR st2c_pow1, st2c_lhs, st2c_rhs : INT;
isfinite : REAL := powf(3);
END_VAR st2c_pow1 := 1; END_PROGRAM`, "st2c_pow1=1\nst2c_lhs=0\nst2c_rhs=0\nisfinite=9\n", "")
}

func TestPowerCallOperands(t *testing.T) {
	code := generateSource(t, `FUNCTION BaseArg : INT BaseArg := 3; END_FUNCTION
FUNCTION ExpArg : REAL ExpArg := 2; END_FUNCTION
PROGRAM P VAR r : REAL; END_VAR r := BaseArg() ** ExpArg(); END_PROGRAM`, 1)
	// Prototypes and definitions use (void), only calls use ().
	if strings.Count(code, "BaseArg()") != 1 || strings.Count(code, "ExpArg()") != 1 ||
		!strings.Contains(code, " = (float)BaseArg(), st2c_rhs = ExpArg(), st2c_pow(") {
		t.Fatalf("unexpected operand evaluation:\n%s", code)
	}
}
