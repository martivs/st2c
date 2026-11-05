(* Один скан: инициализаторы и локальные VAR_TEMP функции, именованные
   и вложенные вызовы, адаптивные литералы, MOD/**, обе конверсии,
   CASE/RETURN/EXIT, три вида циклов, логика с коротким замыканием.
   Имена powf и st2c_lhs проверяют соседство с именами генератора.

   seed=(1/2)^2=0.25, начальный n=16 MOD 5=1, negative=-17 MOD 5=-2.
   unit=2**(1/2)=1: показатель без REAL-контекста, целое 1/2=0.
   Blend(0.5,1,TRUE)=0.25/2=0.125; Blend(2,5,TRUE)=4+0.5=4.5;
   Blend(...,FALSE)=0; вложенный Blend: 1+0.5=1.5, затем 1.5^2/2=1.125.
   powf=0.125+4.5+1.125=5.75 -> rounded=6 (не пограничное округление).
   WHILE: 2^0,2^1,2^2 < 8, на ticks=3 останавливается.
   REPEAT: n=1 -> 2 -> 3 -> 4; FOR: 3+2+1=6, i=1.
   Обе ошибочные степени в safe пропускаются по AND/OR.

   Ожидаемые поля: seed=0.25, n=4, negative=-2, unit=1,
   a=0.125, b=4.5, c=0, nested=1.125, powf=5.75,
   rounded=6, ticks=3, i=1, st2c_lhs=6, safe=1, logic=1. *)
FUNCTION Blend : REAL
VAR_INPUT
    x : REAL;
    n : INT;
    gate : BOOL;
END_VAR
VAR_TEMP
    base : REAL := x ** 2;
    remainder : INT := (+n MOD 5) MOD 3;
    half : REAL := 1 / 2;
END_VAR
IF NOT gate THEN
    RETURN;
END_IF;
REPEAT
    CASE remainder OF
        0: Blend := base + half;
           RETURN;
        1..2: Blend := base / (INT_TO_REAL(remainder) + 1);
              EXIT;
        ELSE Blend := -1;
    END_CASE;
UNTIL TRUE END_REPEAT;
END_FUNCTION

PROGRAM MixedNumeric
VAR
    seed : REAL := (1 / 2) ** 2;
    n : INT := 16 MOD 5;
    negative : INT := -17 MOD 5;
    unit : REAL := 2 ** (1 / 2);
    a, b, c, nested, powf : REAL;
    rounded, ticks, i, st2c_lhs : INT;
    safe, logic : BOOL;
END_VAR
a := Blend(n := n, gate := TRUE, x := seed + 1 / 4);
b := Blend(2, n := 5, gate := TRUE);
c := Blend(1 / 2, n := 7, gate := FALSE);
nested := Blend(Blend(1, n := 5, gate := TRUE), n := 6, gate := TRUE);
powf := a + b + nested;
rounded := REAL_TO_INT(powf);
WHILE 2 ** ticks < 8 DO
    ticks := ticks + 1;
END_WHILE;
REPEAT
    n := n + 1;
UNTIL n MOD 4 = 0 END_REPEAT;
FOR i := 3 TO 1 BY -(1 + n MOD 2) DO
    st2c_lhs := st2c_lhs + i;
END_FOR;
safe := (n = 4 OR 0 ** -1 > 0) AND NOT (n <> 4 & (-2) ** 0.5 > 0);
logic := safe XOR (a > b) OR NOT (ticks = 3);
END_PROGRAM
