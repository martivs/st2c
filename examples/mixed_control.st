(* Три скана: FOR -> REPEAT -> CASE -> WHILE -> CASE внутри ФБ,
   EXIT адресуется каждому из трёх видов цикла. Один экземпляр Sweep
   вызывается дважды за скан: флаги выхода должны создаваться заново.
   FUNCTION FirstSquare возвращается из CASE внутри FOR; PROGRAM тоже
   делает ранний RETURN на скане 2, сохраняя tail для следующего скана.

   Sweep(stop=2): i=1 даёт 2+11=13, i=2 даёт 2+10=12 -> 25.
   Sweep(stop=3): те же 25, затем i=3 даёт 2+101=103 -> 128.
   При j=2, i=2 EXIT пропускает прибавку 1 и выходит из REPEAT.
   При j=3 EXIT выходит из REPEAT без прибавки. EXIT в WHILE оставляет
   REPEAT активным; внешний EXIT после REPEAT покидает FOR.
   FirstSquare пропускает кратные 3: для целей 1/4/9 ответ 1/2/4.

   Поле  | Скан 1 | Скан 2 | Скан 3
   scan  | 1      | 2      | 3
   first | 25     | 128    | 25
   again | 25     | 25     | 25
   total | 50     | 203    | 253
   found | 1      | 2      | 4
   tail  | 1      | 1      | 2
   scaled| 6.25   | 32     | 6.25
   valid | 1      | 1      | 1 *)
FUNCTION StopAt : INT
VAR_INPUT n : INT; END_VAR
IF n MOD 2 = 0 THEN
    StopAt := 3;
ELSIF n > 0 THEN
    StopAt := 2;
ELSE
    StopAt := 1;
END_IF;
END_FUNCTION

FUNCTION FirstSquare : INT
VAR_INPUT target : REAL; END_VAR
VAR_TEMP k : INT; END_VAR
FOR k := 1 TO 5 DO
    CASE k MOD 3 OF
        0:
        ELSE
            IF k ** 2 >= target THEN
                FirstSquare := k;
                RETURN;
            END_IF;
    END_CASE;
END_FOR;
END_FUNCTION

FUNCTION_BLOCK Sweep
VAR_INPUT stop : INT; END_VAR
VAR_OUTPUT count : INT; END_VAR
VAR i, j : INT; END_VAR
count := 0;
FOR i := 1 TO 3 DO
    j := 0;
    REPEAT
        j := j + 1;
        CASE j MOD 3 OF
            1: count := count + 1;
            2: WHILE TRUE DO
                   CASE i OF
                       1..2: count := count + 10;
                             EXIT;
                       ELSE count := count + 100;
                            EXIT;
                   END_CASE;
               END_WHILE;
               CASE i OF
                   2: EXIT;
               END_CASE;
            ELSE EXIT;
        END_CASE;
        count := count + 1;
    UNTIL j >= stop END_REPEAT;
    IF i >= stop THEN
        EXIT;
    END_IF;
END_FOR;
END_FUNCTION_BLOCK

PROGRAM MixedControl
VAR
    scan, first, again, total, found, tail : INT;
    scaled : REAL;
    valid : BOOL;
    worker : Sweep;
END_VAR
scan := scan + 1;
worker(stop := StopAt(scan), count => first);
worker(stop := 2, count => again);
total := total + first + again;
found := FirstSquare(scan ** 2);
scaled := INT_TO_REAL(first) / 4;
valid := again = 25 AND found > 0 AND NOT (found MOD 3 = 0);
CASE scan OF
    2: RETURN;
END_CASE;
tail := tail + 1;
END_PROGRAM
