PROGRAM NestedForInIf
VAR
    x : INT;
    i : INT;
    total : INT;
END_VAR

x := 7;
total := 0;

IF x > 5 THEN
    FOR i := 1 TO x DO
        total := total + i;
    END_FOR;
ELSE
    total := 0;
END_IF;

END_PROGRAM
