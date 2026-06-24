PROGRAM Example
VAR
    x : INT;
    y : INT;
    sum : INT;
END_VAR

x := 10;
y := 3;
sum := 0;

FOR x := 1 TO 5 DO
    sum := sum + x;
END_FOR;

IF sum > 10 THEN
    y := 1;
ELSE
    y := 0;
END_IF;

END_PROGRAM
