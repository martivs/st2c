PROGRAM DeeplyNested
VAR
    i : INT;
    j : INT;
    acc : INT;
END_VAR

acc := 0;

FOR i := 1 TO 3 DO
    FOR j := 1 TO 3 DO
        IF (i + j) > 4 THEN
            IF i = j THEN
                acc := acc + 1;
            ELSE
                acc := acc + 2;
            END_IF;
        ELSE
            acc := acc - 1;
        END_IF;
    END_FOR;
END_FOR;

END_PROGRAM
