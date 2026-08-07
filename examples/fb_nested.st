(* Вложенность функциональных блоков: экземпляр Accum внутри Machine.

   Проверяет две вещи, которые ломаются независимо друг от друга:
     - Machine_init обязан рекурсивно вызвать Accum_init, иначе seed
       проинициализируется, а состояние вложенного экземпляра — нет;
     - typedef Accum обязан идти в файле раньше typedef Machine: Machine
       содержит Accum по значению, и прототипы этого не спасают — нужна
       топологическая сортировка типов по вложенности.

   Ожидание по сканам: doubled = 110, 120, 130 (inner.sum растёт на 5,
   удваивается, плюс seed = 100).

   ВНИМАНИЕ: разбирается начиная с этапа 3, исполняется с этапа 7. *)

FUNCTION_BLOCK Accum
VAR_INPUT
    add : INT;
END_VAR
VAR_OUTPUT
    sum : INT;
END_VAR
    sum := sum + add;
END_FUNCTION_BLOCK

FUNCTION_BLOCK Machine
VAR_INPUT
    tick : INT;
END_VAR
VAR_OUTPUT
    doubled : INT;
END_VAR
VAR
    inner : Accum;
    seed : INT := 100;
END_VAR
    inner(add := tick);
    doubled := inner.sum * 2 + seed;
END_FUNCTION_BLOCK

PROGRAM FbNested
VAR
    m : Machine;
    result : INT;
END_VAR

m(tick := 5, doubled => result);

END_PROGRAM
