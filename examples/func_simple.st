(* Функции: VAR_INPUT как параметры, возврат присваиванием имени функции,
   вложенные вызовы, позиционные и именованные аргументы.

   Clamp дополнительно проверяет, что возвратной переменной можно присваивать
   несколько раз и что внутри функции работают операторы.

   ВНИМАНИЕ: разбирается начиная с этапа 3, исполняется с этапа 5.
   До этапа 3 `go run ./src examples/func_simple.st` завершается ошибкой
   разбора — это ожидаемо. *)

FUNCTION Add : INT
VAR_INPUT
    x : INT;
    y : INT;
END_VAR
    Add := x + y;
END_FUNCTION

FUNCTION Clamp : INT
VAR_INPUT
    v : INT;
    lo : INT;
    hi : INT;
END_VAR
    Clamp := v;
    IF v < lo THEN Clamp := lo; END_IF
    IF v > hi THEN Clamp := hi; END_IF
END_FUNCTION

PROGRAM FuncSimple
VAR
    plain : INT;
    nested : INT;
    named : INT;
    clamped : INT;
END_VAR

plain := Add(2, 3);                              // 5
nested := Add(Add(1, 2), Add(3, 4));             // 3 + 7 = 10
named := Add(x := 10, y := 5);                   // 15
clamped := Clamp(v := 99, lo := 0, hi := 50);    // 50

END_PROGRAM
