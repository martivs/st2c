(* Три скана. Choose берёт +n MOD 3 = 1/2/0: первые два раза
   возвращает квадрат 1/4 прямо из CASE, затем ELSE возвращает 0.5.
   ФБ прибавляет эти значения к состоянию. Во вложенном REPEAT на
   первом обороте FOR работает только EXIT внутреннего цикла;
   на втором CASE передаёт EXIT внешнему FOR до прибавки 10.
   Поля PROGRAM: scan=1/2/3, value=1/4/0.5, total=1/5/5.5,
   i=2/2/2, marks=11/11/11. *)
FUNCTION Choose : REAL
VAR_INPUT n : INT; END_VAR
CASE +n MOD 3 OF
    1..2: Choose := INT_TO_REAL(n) ** 2;
          RETURN;
    ELSE Choose := 0.5;
END_CASE;
END_FUNCTION

FUNCTION_BLOCK Accumulate
VAR_INPUT value : REAL; END_VAR
VAR_OUTPUT total : REAL; END_VAR
total := total + value;
END_FUNCTION_BLOCK

PROGRAM OpsAll
VAR
    counter : Accumulate;
    scan : INT;
    value, total : REAL;
    i, marks : INT;
END_VAR
scan := scan + 1;
value := Choose(scan);
counter(value := value, total => total);
marks := 0;
FOR i := 1 TO 3 DO
    CASE i OF
        1: REPEAT
               marks := marks + 1;
               EXIT;
           UNTIL TRUE END_REPEAT;
        2: EXIT;
    END_CASE;
    marks := marks + 10;
END_FOR;
END_PROGRAM
