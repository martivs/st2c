(* Все формы объявлений: списки имён, инициализаторы, квалификаторы
   CONSTANT/RETAIN, разные виды VAR-блоков.

   Проверяет раскладку struct состояния и функцию _init: переменная без
   инициализатора обязана обнуляться явно, а инициализатор в объявлении со
   списком имён — применяться ко всем именам сразу. *)
PROGRAM VarsAll
VAR CONSTANT
    LIMIT : INT := 100;
    STEP_SIZE : INT := 3;
END_VAR
VAR RETAIN
    saved : INT := -5;
END_VAR
VAR_INPUT
    a, b, c : INT := 7;      // один инициализатор на три имени
END_VAR
VAR_OUTPUT
    result : INT;
END_VAR
VAR
    scratch : INT;
    untouched : INT;         // без инициализатора — должен стать 0
END_VAR

scratch := a + b + c;                             // 21
saved := saved + STEP_SIZE;                       // -5 + 3 = -2
result := scratch + saved + LIMIT + untouched;    // 21 - 2 + 100 + 0 = 119

END_PROGRAM
