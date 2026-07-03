PROGRAM NestedIfInFor
VAR
    i : INT;
    sum : INT;
    flag : INT;
END_VAR

sum := 0;
flag := 0;

FOR i := 1 TO 10 DO
    IF i > 5 THEN
        sum := sum + i;
    ELSE
        flag := flag + 1;
    END_IF;
END_FOR;

END_PROGRAM
