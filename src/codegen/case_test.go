package codegen_test

import (
	"fmt"
	"strings"
	"testing"
)

func TestCaseExitPaths(t *testing.T) {
	for _, loop := range []struct{ name, open, close string }{
		{"for", "FOR i := 1 TO 4 DO", "END_FOR"},
		{"while", "WHILE TRUE DO", "END_WHILE"},
		{"repeat", "REPEAT", "UNTIL FALSE END_REPEAT"},
	} {
		for _, depth := range []int{1, 2, 4} {
			t.Run(fmt.Sprintf("%s/%d", loop.name, depth), func(t *testing.T) {
				body := "IF n = pass THEN EXIT; missed := 100; END_IF;"
				// Wrap the exit in alternating point/default and range arms.
				for level := 0; level < depth; level++ {
					if level%2 == 0 {
						body = "CASE n OF 0: ELSE " + body + " tail := tail + 1; END_CASE;"
					} else {
						body = "CASE n OF 1..4: " + body + " tail := tail + 1; END_CASE;"
					}
				}
				src := "PROGRAM P VAR pass, n, i, missed, tail : INT; END_VAR " +
					"FOR pass := 1 TO 2 DO n := 0; " + loop.open + " n := n + 1; " +
					"IF TRUE THEN " + body + " END_IF; tail := tail + 1; " +
					loop.close + "; END_FOR; END_PROGRAM"
				code := generateSource(t, src, 2)
				if strings.Count(code, "int st2c_exit") != 1 || strings.Count(code, "if (st2c_exit) break;") != depth {
					t.Fatal("expected one loop flag and a check after each crossed switch")
				}
				i := 0
				if loop.name == "for" {
					i = 2
				}
				// Only iteration 1 of pass 2 reaches any tail.
				scan := "pass=2\nn=2\ni=%d\nmissed=0\ntail=%d\n"
				runGeneratedC(t, code, fmt.Sprintf(scan, i, depth+1)+fmt.Sprintf(scan, i, 2*(depth+1)))
			})
		}
	}
}

func TestCaseInnerLoopExit(t *testing.T) {
	for _, tc := range []struct {
		open, close string
		i           int
	}{
		{"FOR i := 1 TO 3 DO", "END_FOR", 1},
		{"WHILE TRUE DO", "END_WHILE", 0},
		{"REPEAT", "UNTIL FALSE END_REPEAT", 0},
	} {
		for _, flagged := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/flag=%t", strings.Fields(tc.open)[0], flagged), func(t *testing.T) {
				exit := "EXIT;"
				if flagged {
					exit = "CASE 1 OF 1: EXIT; END_CASE;"
				}
				code := generateSource(t, "PROGRAM P VAR j,i,hits,tail : INT; END_VAR "+
					"FOR j := 1 TO 4 DO CASE j OF 1..2: "+tc.open+" hits := hits + 1; "+
					exit+" hits := 100; "+tc.close+"; tail := tail + 1; "+
					"ELSE EXIT; END_CASE; tail := tail + 10; END_FOR; END_PROGRAM", 1)
				flags := 1 // outer loop's EXIT from ELSE
				if flagged {
					flags++
				}
				if strings.Count(code, "int st2c_exit") != flags {
					t.Fatal("wrong number of loop flags")
				}
				runGeneratedC(t, code, fmt.Sprintf("j=3\ni=%d\nhits=2\ntail=22\n", tc.i))
			})
		}
	}
}

func TestCaseAroundLoops(t *testing.T) {
	code := generateSource(t, `PROGRAM P VAR i, hits : INT; END_VAR
CASE 1 OF
 1: FOR i := 1 TO 5 DO hits := hits + 1; EXIT; hits := 99; END_FOR;
    WHILE TRUE DO hits := hits + 1; EXIT; hits := 99; END_WHILE;
    REPEAT hits := hits + 1; EXIT; hits := 99; UNTIL FALSE END_REPEAT;
    hits := hits + 10;
END_CASE;
END_PROGRAM`, 1)
	if strings.Contains(code, "st2c_exit") {
		t.Fatal("EXIT targets the inner loop directly")
	}
	runGeneratedC(t, code, "i=1\nhits=13\n")
}

func TestCaseBounds(t *testing.T) {
	code := generateSource(t, `FUNCTION Band : INT VAR_INPUT v : INT; END_VAR
CASE v OF
 -03..-01, +0008: Band := 1;
 1, 3..5: Band := 2;
 10:
 ELSE Band := -1;
END_CASE;
END_FUNCTION
PROGRAM P
VAR lo, mid, hi, below, above, point, empty, miss, full, oct : INT; END_VAR
lo := Band(-3); mid := Band(4); hi := Band(-1);
below := Band(-4); above := Band(6); point := Band(8); empty := Band(10);
CASE 2 OF 1: miss := 99; END_CASE;
CASE 2 OF -8..-3: miss := 98; ELSE END_CASE;
CASE -32768 OF -32768..32767: full := 1; END_CASE;
CASE 32767 OF -32768..32767: full := full + 1; END_CASE;
CASE 8 OF +0008, 09: oct := 8; END_CASE;
END_PROGRAM`, 1)
	// Three Band arms, two unmatched CASEs, two full ranges and two point labels.
	if strings.Count(code, "case ") != 9 {
		t.Fatal("unexpected number of switch labels")
	}
	if strings.Contains(code, "st2c_exit") {
		t.Fatal("CASE without EXIT needs no flag")
	}
	runGeneratedC(t, code, "lo=1\nmid=2\nhi=1\nbelow=-1\nabove=-1\npoint=1\nempty=0\nmiss=0\nfull=2\noct=8\n")
}

func TestCaseSelectorsNamesAndReturn(t *testing.T) {
	code := generateSource(t, `FUNCTION st2c_sel : INT
VAR_INPUT st2c_sel1 : INT; END_VAR
VAR st2c_arm, st2c_exit : INT; END_VAR
WHILE TRUE DO
 CASE st2c_sel1 OF 1..2: st2c_sel := st2c_sel1; RETURN;
 ELSE st2c_arm := 3; st2c_exit := 4; EXIT;
 END_CASE;
 st2c_sel := 99;
END_WHILE;
st2c_sel := st2c_arm + st2c_exit;
END_FUNCTION
FUNCTION_BLOCK Meter
VAR_INPUT n : INT; END_VAR VAR_OUTPUT total : INT; END_VAR
CASE st2c_sel(n) OF
 1..2: total := total + n; RETURN;
 ELSE total := total + 10;
END_CASE;
total := total + 1;
END_FUNCTION_BLOCK
PROGRAM P
VAR m : Meter; scan, seen, tail, st2c_exit1, st2c_arm1 : INT; END_VAR
scan := scan + 1;
m(n := scan, total => seen);
IF TRUE THEN CASE m.total OF 3..5: RETURN; END_CASE; END_IF;
tail := tail + 1;
END_PROGRAM`, 3)
	for _, fragment := range []string{"st2c_sel(self->n)", "int32_t st2c_sel4 = self->m.total;", "int st2c_exit2 = 0;", "int st2c_arm2 = 0;", "int32_t st2c_sel2 = st2c_sel1;"} {
		if strings.Count(code, fragment) != 1 {
			t.Fatalf("expected %q exactly once in generated code", fragment)
		}
	}
	runGeneratedC(t, code, "scan=1\nseen=1\ntail=1\nst2c_exit1=0\nst2c_arm1=0\n"+
		"scan=2\nseen=3\ntail=1\nst2c_exit1=0\nst2c_arm1=0\n"+
		"scan=3\nseen=14\ntail=2\nst2c_exit1=0\nst2c_arm1=0\n")
}
