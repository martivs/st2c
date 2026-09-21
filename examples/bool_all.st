(* Тип BOOL и логические операции: объявления с инициализаторами TRUE/FALSE,
   присваивание результата сравнения, AND / OR / XOR / NOT и синоним &,
   приоритеты по IEC (OR < XOR < AND < сравнения; NOT сильнее любого
   бинарного), IF по переменной BOOL, по NOT и по связке сравнений без
   скобок, равенство и неравенство BOOL, BOOL как вход и возврат FUNCTION,
   как вход и выход FUNCTION_BLOCK с состоянием (SR-защёлка и T-триггер),
   флаги, накапливаемые в целочисленном FOR, BOOL в состоянии PROGRAM.

   Драйвер печатает BOOL через %d: 1 — TRUE, 0 — FALSE.

   Состояние защёлки и триггера видно только за несколько сканов, поэтому
   проверять так:
       go run ./src -main -scans 3 -o /tmp/a.c examples/bool_all.st
   Ожидание (три скана; поля, не зависящие от скана, одинаковы в каждом):
   on=1 off=0 n=7 gt=1 both=1 either=1 differ=0 neither=1 prec=1 grouped=0
   chain=1 cmp=55 inside=1 gated=7 i=10 seen=1 allSmall=0; по сканам:
   cycle = 1, 2, 3; armed = 1, 0, 1; held = 1, 1, 0; flipped = 1, 0, 1.
   Расчёт — в комментариях у операторов. *)

FUNCTION InRange : BOOL
VAR_INPUT
    v : INT;
    lo : INT;
    hi : INT;
END_VAR
    InRange := v >= lo AND v <= hi;     // AND слабее отношений: (v >= lo) AND (v <= hi)
END_FUNCTION

FUNCTION Gate : INT
VAR_INPUT
    en : BOOL;
    v : INT;
END_VAR
    Gate := 0;
    IF en THEN Gate := v; END_IF        // условие — BOOL-параметр
END_FUNCTION

FUNCTION_BLOCK Latch                    // SR-защёлка: set устанавливает, rst сбрасывает
VAR_INPUT
    set : BOOL;
    rst : BOOL;
END_VAR
VAR_OUTPUT
    q : BOOL;
END_VAR
    q := (q OR set) AND NOT rst;        // q переживает вызов: без входов держит значение
END_FUNCTION_BLOCK

FUNCTION_BLOCK Toggle                   // T-триггер: каждый вызов с t = TRUE переключает q
VAR_INPUT
    t : BOOL := TRUE;                   // инициализатор входа: вызов без t переключает
END_VAR
VAR_OUTPUT
    q : BOOL;
END_VAR
    q := q XOR t;
END_FUNCTION_BLOCK

PROGRAM BoolAll
VAR
    on : BOOL := TRUE;
    off : BOOL := FALSE;
    n : INT := 7;
    gt : BOOL;                          // результат сравнения
    both : BOOL;                        // AND
    either : BOOL;                      // OR
    differ : BOOL;                      // XOR
    neither : BOOL;                     // NOT и &
    prec : BOOL;                        // a OR b AND c
    grouped : BOOL;                     // (a OR b) AND c
    chain : BOOL;                       // a OR b XOR c AND d
    cmp : INT;                          // условия IF
    inside : BOOL;                      // BOOL-возврат функции
    gated : INT;                        // BOOL-вход функции
    i : INT;
    seen : BOOL;                        // флаг «встретилось» из FOR (OR)
    allSmall : BOOL;                    // флаг «все» из FOR (AND)
    cycle : INT;                        // номер скана
    armed : BOOL;                       // BOOL в состоянии PROGRAM
    held : BOOL;                        // выход защёлки
    flipped : BOOL;                     // выход триггера
    sr : Latch;
    flip : Toggle;
END_VAR

gt := n > 5;                            // 7 > 5 — да: 1
both := on AND gt;                      // TRUE AND TRUE: 1
either := off OR gt;                    // FALSE OR TRUE: 1
differ := on XOR gt;                    // TRUE XOR TRUE: 0
neither := NOT off & NOT differ;        // NOT сильнее &: TRUE & TRUE: 1

prec := on OR off AND off;              // AND сильнее OR: TRUE OR (FALSE AND FALSE): 1
grouped := (on OR off) AND off;         // скобки: (TRUE) AND FALSE: 0
chain := on OR off XOR gt AND both;     // OR < XOR < AND: TRUE OR (FALSE XOR (TRUE AND TRUE)): 1
                                        // (слева направо было бы ((TRUE OR FALSE) XOR TRUE) AND TRUE = 0)

cmp := 0;
IF on THEN cmp := cmp + 1; END_IF                   // переменная BOOL — да: 1
IF NOT off THEN cmp := cmp + 2; END_IF              // NOT — да: 3
IF n > 5 AND n < 10 THEN cmp := cmp + 4; END_IF     // связка сравнений без скобок — да: 7
IF differ OR n = 0 THEN cmp := cmp + 8; END_IF      // OR слабее =: FALSE OR FALSE — нет: 7
IF gt = on THEN cmp := cmp + 16; END_IF             // равенство BOOL: TRUE = TRUE — да: 23
IF gt <> off THEN cmp := cmp + 32; END_IF           // неравенство BOOL: TRUE <> FALSE — да: 55

inside := InRange(n, 1, 10);                        // 1 <= 7 <= 10 — да: 1
gated := Gate(inside, n) + Gate(en := NOT inside, v := 100);   // 7 + 0 = 7

seen := FALSE;
allSmall := TRUE;
FOR i := 1 TO 10 DO
    seen := seen OR i = n;              // OR слабее =: встретилось ли i = 7 — да: 1
    allSmall := allSmall AND i < 5;     // все ли i < 5 — нет (с i = 5): 0
END_FOR                                 // i = 10 после цикла

cycle := cycle + 1;                     // 1, 2, 3
armed := NOT armed;                     // состояние PROGRAM: 1, 0, 1

sr(set := cycle = 1, rst := cycle = 3, q => held);    // скан 1: (0 OR 1) AND NOT 0 = 1;
                                                      // скан 2: входы FALSE, держит: 1;
                                                      // скан 3: rst — (1 OR 0) AND NOT 1 = 0
flip(q => flipped);                     // t = TRUE из инициализатора: 0 XOR 1 = 1, 1 XOR 1 = 0, 0 XOR 1 = 1

END_PROGRAM
