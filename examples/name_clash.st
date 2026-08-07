(* Имена ST, конфликтующие с именами в сгенерированном C.

   Две разные опасности:
     1. Ключевые слова C, не являющиеся ключевыми словами ST: switch, default,
        static, struct. (int и for сюда не годятся — они ключевые слова и в
        ST, так что переменными быть не могут.)
     2. Служебные имена, которые codegen порождает сам: self — параметр
        функции шага, __end и __step — временные цикла FOR.

   Вторая опаснее: коллизия здесь не даёт ошибки компиляции, а молча меняет
   поведение программы. Цикл в конце ловит именно это — если временная FOR
   затрёт переменную __step, сумма получится другой. *)
PROGRAM NameClash
VAR
    switch, default, static : INT;
    struct : INT := 1;
    self : INT := 2;
    __end : INT := 3;
    __step : INT := 4;
    i : INT;
    total : INT;
END_VAR

switch := 10;
default := 20;
static := 30;

total := switch + default + static + struct + self + __end + __step;   // 70

FOR i := 1 TO 3 DO
    total := total + __step;    // 70 + 4 * 3 = 82
END_FOR

END_PROGRAM
