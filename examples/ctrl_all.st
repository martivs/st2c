(* Управляющие конструкции: цепочка ELSIF (первая истинная ветка, ветка без
   ELSE, вложенный IF внутри ELSIF, старая форма ELSE IF рядом с ELSIF,
   условие-член ФБ и условие-вызов функции, EXIT внутри ветки ELSIF),
   WHILE (ноль итераций, несколько итераций, WHILE TRUE с EXIT, FOR внутри
   WHILE, WHILE внутри WHILE, условие с REAL и адаптивным литералом, условие
   с NOT и связкой, условие-BOOL-переменная), REPEAT (ровно одна итерация при
   истинном UNTIL, EXIT из REPEAT, REPEAT внутри REPEAT, REPEAT в FUNCTION,
   условие-вызов функции в UNTIL), EXIT (из FOR со значением переменной цикла,
   из вложенного цикла — только внутренний, из цикла в теле FUNCTION и в теле
   FUNCTION_BLOCK), WHILE в теле FUNCTION_BLOCK за три скана, END_WHILE и
   END_REPEAT как с необязательной `;`, так и без неё.

   Проверять так:
       go run ./src -main -scans 3 -o /tmp/a.c examples/ctrl_all.st
   Ожидание (три скана; поля, не зависящие от скана, одинаковы в каждом):
   t=5 first=1 band=99 sign=2 legacy=2 whileZero=0 repeatOne=1 sum=15 k=3
   rep=3 found=5 i=3 j=3 outerCnt=3 innerCnt=6 spins=4 acc=10 r=0.625
   halves=4 flag=1 tries=4 digitsZero=1 digitsBig=5 pick=3 taps=11 wOuter=4
   wInner=12 wStep=3 rOuter=3 rInner=9 rStep=3 stepsUsed=6 grown=4 armed=0
   paces=3; по сканам:
   cycle = 1, 2, 3; mode = 10, 20, 30; lvl = 7, 10, 9; drained = 2, 3, 4;
   rampSpeed = 6, 12, 15; rampTicks = 2, 2, 1; rampTop = 0, 1, 1;
   gear = 2, 1, 1.
   Расчёт — в комментариях у операторов; у каждого блока указано, какое число
   даст типичная ошибка реализации (проверено на ручной C-трансляции,
   gcc -std=c99 -Wall -Wextra: каждая такая ошибка меняет число, а не вешает
   тест; исключения — две ошибки, вешающие цикл: REPEAT без отрицания в C и
   не сработавший EXIT при WHILE TRUE, их ловит таймаут). *)

FUNCTION Digits : INT                   // REPEAT в FUNCTION: число десятичных цифр
VAR_INPUT
    v : INT;
END_VAR
VAR
    n : INT;
    c : INT;
END_VAR
    n := v;
    c := 0;
    REPEAT                              // тело хотя бы раз: у нуля одна цифра
        c := c + 1;
        n := n / 10;
    UNTIL n = 0
    END_REPEAT;
    Digits := c;                        // Digits(0) = 1 (как while — дал бы 0); Digits(12345) = 5
END_FUNCTION

FUNCTION_BLOCK Drain                    // WHILE в ФБ: долить inflow, сливать по 4, пока level > 10
VAR_INPUT
    inflow : INT;
END_VAR
VAR_OUTPUT
    level : INT;                        // переживает вызов — состояние ФБ
    steps : INT;                        // сколько порций слито за этот вызов
END_VAR
    level := level + inflow;
    steps := 0;
    WHILE level > 10 DO
        level := level - 4;
        steps := steps + 1;
    END_WHILE;
END_FUNCTION_BLOCK

FUNCTION StepsTo : INT                  // EXIT из WHILE в теле FUNCTION:
VAR_INPUT                               // сколько слагаемых 1+2+3+… нужно,
    limit : INT;                        // чтобы сумма превысила limit
END_VAR
VAR
    n : INT;
    acc : INT;
END_VAR
    WHILE n < 100 DO                    // граница 100 — страховка, выход по EXIT
        n := n + 1;
        acc := acc + n;
        IF acc > limit THEN
            StepsTo := n;
            EXIT;
        END_IF;
    END_WHILE                           // `;` после END_WHILE опущена (решение 6)
END_FUNCTION                            // StepsTo(20): 1+2+3+4+5+6=21 > 20 → 6
                                        // (не сработал EXIT — 100: присваивание
                                        //  повторится до границы страховки)

FUNCTION Ripe : BOOL                    // BOOL-возврат для условий UNTIL и ELSIF
VAR_INPUT
    v : INT;
END_VAR
    Ripe := v >= 4;
END_FUNCTION

FUNCTION_BLOCK Ramp                     // EXIT из REPEAT в теле ФБ: разгон не
VAR_INPUT                               // быстрее двух шагов за вызов
    target : INT;
END_VAR
VAR_OUTPUT
    speed : INT;                        // переживает вызов — состояние ФБ
    ticks : INT;                        // шагов разгона за этот вызов
    atTop : BOOL;                       // цель достигнута
END_VAR
    ticks := 0;
    REPEAT                              // тело хотя бы раз: ticks ещё 0
        IF ticks >= 2 THEN EXIT; END_IF;
        speed := speed + 3;
        ticks := ticks + 1;
    UNTIL speed >= target
    END_REPEAT                          // `;` после END_REPEAT опущена (решение 6)
    atTop := speed >= target;
END_FUNCTION_BLOCK

PROGRAM CtrlAll
VAR
    cycle : INT;                        // номер скана
    t : INT := 5;
    first : INT;                        // ELSIF: первая истинная ветка
    mode : INT;                         // ELSIF по номеру скана: каждая ветка и ELSE
    band : INT := 99;                   // ELSIF без ELSE, ни одна ветка не истинна
    sign : INT;                         // вложенный IF внутри ELSIF
    legacy : INT;                       // старая форма ELSE IF ... END_IF END_IF
    whileZero : INT;                    // WHILE, ложный на входе
    repeatOne : INT;                    // REPEAT, UNTIL истинен после первой итерации
    sum : INT;                          // WHILE с итерациями
    k : INT;
    rep : INT;                          // EXIT из REPEAT
    found : INT;                        // EXIT из FOR
    i : INT;
    j : INT;
    outerCnt : INT;                     // EXIT из вложенного цикла
    innerCnt : INT;
    spins : INT;                        // WHILE TRUE ... EXIT
    acc : INT;                          // FOR внутри WHILE
    r : REAL;                           // условие WHILE с REAL
    halves : INT;
    flag : BOOL;                        // условие WHILE с NOT и связкой
    tries : INT;
    digitsZero : INT;                   // REPEAT в FUNCTION
    digitsBig : INT;
    tank : Drain;                       // WHILE в ФБ; драйвер экземпляр не печатает
    lvl : INT;
    drained : INT;
    pick : INT;                         // EXIT внутри ветки ELSIF
    taps : INT;
    wOuter : INT;                       // EXIT из WHILE, вложенного в WHILE
    wInner : INT;
    wStep : INT;
    rOuter : INT;                       // EXIT из REPEAT, вложенного в REPEAT
    rInner : INT;
    rStep : INT;
    stepsUsed : INT;                    // EXIT в цикле внутри FUNCTION
    grown : INT;                        // UNTIL с условием-вызовом функции
    armed : BOOL;                       // WHILE с условием-BOOL-переменной
    paces : INT;
    ramp : Ramp;                        // EXIT в REPEAT в теле ФБ
    rampSpeed : INT;
    rampTicks : INT;
    rampTop : BOOL;
    gear : INT;                         // ELSIF по члену ФБ и по вызову функции
END_VAR
    cycle := cycle + 1;

    // ---- ELSIF ----

    // t = 5: истинны все три условия, выбирается ПЕРВАЯ ветка → first = 1.
    // (Если ветки не связаны через else — first = 2 или 3.)
    IF t < 10 THEN first := 1;
    ELSIF t < 20 THEN first := 2;
    ELSIF t < 30 THEN first := 3;
    END_IF;

    // Скан 1 — первая ветка, скан 2 — ELSIF, скан 3 — ELSE: mode = 10, 20, 30.
    IF cycle = 1 THEN mode := 10;
    ELSIF cycle = 2 THEN mode := 20;
    ELSE mode := 30;
    END_IF;

    // Ни одно условие не истинно, ELSE нет — band остаётся 99 из инициализатора.
    IF t > 100 THEN band := 1;
    ELSIF t > 50 THEN band := 2;
    END_IF;

    // t = 5: первая ветка ложна, ELSIF истинна, внутри неё t > 3 → sign = 2.
    IF t > 10 THEN sign := 1;
    ELSIF t > 0 THEN
        IF t > 3 THEN sign := 2; ELSE sign := 3; END_IF;
    ELSE sign := 4;
    END_IF;

    // Старая форма — вложенный IF в ELSE — работает как прежде: legacy = 2.
    // В AST это If внутри Else, а не ElsIf (решение 1 плана).
    IF t > 10 THEN legacy := 1;
    ELSE
        IF t > 3 THEN legacy := 2; END_IF;
    END_IF;

    // ---- WHILE и REPEAT: ноль итераций против одной ----

    // Условие ложно на входе — тело не выполняется: whileZero = 0.
    // (Если WHILE сгенерирован как do-while — whileZero = 1.)
    whileZero := 0;
    k := 0;
    WHILE k > 0 DO
        whileZero := whileZero + 1;
        k := k - 1;
    END_WHILE;

    // UNTIL истинен уже после первой итерации (1 < 100) — ровно одна: repeatOne = 1.
    // (Если в C забыто отрицание !(...) — цикл крутится, пока repeatOne < 100:
    // repeatOne = 100.)
    repeatOne := 0;
    REPEAT
        repeatOne := repeatOne + 1;
    UNTIL repeatOne < 100
    END_REPEAT;

    // ---- WHILE с итерациями ----

    // k = 1..5, sum = 1 + 2 + 3 + 4 + 5 = 15, k = 5 (позже перезаписывается).
    sum := 0;
    k := 0;
    WHILE k < 5 DO
        k := k + 1;
        sum := sum + k;
    END_WHILE;

    // ---- EXIT ----

    // EXIT из REPEAT: rep = 1, 2, 3 → на 3 выход, rep = 3. (Без EXIT — 10.)
    rep := 0;
    REPEAT
        rep := rep + 1;
        IF rep = 3 THEN EXIT; END_IF;
    UNTIL rep >= 10
    END_REPEAT;

    // EXIT из FOR: первое i с i*i > 20 — это 5 (25), found = 5; переменная
    // цикла в момент выхода тоже 5 (позже перезаписывается следующим FOR).
    // (Без EXIT присваивание повторилось бы до i = 10: found = 10.)
    found := 0;
    FOR i := 1 TO 10 DO
        IF i * i > 20 THEN
            found := i;
            EXIT;
        END_IF;
    END_FOR;

    // EXIT во внутреннем FOR выходит только из внутреннего: на каждой из трёх
    // внешних итераций внутренний делает j = 1, 2 и выходит на j = 3.
    // outerCnt = 3, innerCnt = 3 * 2 = 6; после цикла i = 3, j = 3.
    // (Если EXIT выбил бы и внешний цикл — outerCnt = 1, innerCnt = 2.)
    outerCnt := 0;
    innerCnt := 0;
    FOR i := 1 TO 3 DO
        outerCnt := outerCnt + 1;
        FOR j := 1 TO 10 DO
            IF j > 2 THEN EXIT; END_IF;
            innerCnt := innerCnt + 1;
        END_FOR;
    END_FOR;

    // WHILE TRUE с выходом по EXIT: spins = 1, 2, 3, 4 → выход, spins = 4.
    spins := 0;
    WHILE TRUE DO
        spins := spins + 1;
        IF spins >= 4 THEN EXIT; END_IF;
    END_WHILE;

    // ---- FOR внутри WHILE ----

    // k = 1: acc += 1 → 1; k = 2: += 1 + 2 → 4; k = 3: += 1 + 2 + 3 → 10.
    // acc = 10, k = 3, j = 3 (последний FOR шёл 1..3).
    acc := 0;
    k := 0;
    WHILE k < 3 DO
        k := k + 1;
        FOR j := 1 TO k DO
            acc := acc + j;
        END_FOR;
    END_WHILE;

    // ---- Условия разных типов ----

    // REAL и адаптивный литерал (1 в контексте REAL): 10 → 5 → 2.5 → 1.25 →
    // 0.625; halves = 4, r = 0.625 (точно в двоичном представлении).
    r := 10.0;
    halves := 0;
    WHILE r > 1 DO
        r := r / 2;
        halves := halves + 1;
    END_WHILE;

    // NOT и связка: tries*3 = 3, 6, 9, 12 — на 4-й попытке 12 > 10, flag = TRUE.
    // tries = 4, flag = 1.
    flag := FALSE;
    tries := 0;
    WHILE NOT flag AND tries < 10 DO
        tries := tries + 1;
        flag := tries * 3 > 10;
    END_WHILE;

    // ---- REPEAT в FUNCTION ----

    digitsZero := Digits(0);            // 1
    digitsBig := Digits(12345);         // 5

    // ---- WHILE в ФБ, состояние между сканами ----

    // Скан 1: level 0 + 15 = 15 → 11 → 7; lvl = 7, drained = 2.
    // Скан 2: 7 + 15 = 22 → 18 → 14 → 10; lvl = 10, drained = 3 (10 > 10 ложно).
    // Скан 3: 10 + 15 = 25 → 21 → 17 → 13 → 9; lvl = 9, drained = 4.
    tank(inflow := 15, level => lvl, steps => drained);

    // ---- EXIT внутри ветки ELSIF ----

    // pick = 1 → первая ветка (taps = 1); pick = 2 → первая ELSIF (taps = 11);
    // pick = 3 → вторая ELSIF: EXIT выходит из WHILE. pick = 3, taps = 11.
    // (Если EXIT внутри ELSIF не срабатывает — цикл дойдёт до pick = 9 и
    //  ветка ELSE добавит 100 шесть раз: taps = 611.)
    pick := 0;
    taps := 0;
    WHILE pick < 9 DO
        pick := pick + 1;
        IF pick = 1 THEN taps := taps + 1;
        ELSIF pick = 2 THEN taps := taps + 10;
        ELSIF pick = 3 THEN EXIT;
        ELSE taps := taps + 100;
        END_IF;
    END_WHILE                           // `;` опущена (решение 6)

    // ---- EXIT из WHILE, вложенного в WHILE ----

    // Внешний WHILE — четыре итерации; внутренний на каждой делает wStep
    // 1, 2, 3 и выходит по EXIT на третьем. wOuter = 4, wInner = 4 * 3 = 12,
    // wStep = 3. (Если EXIT выбил бы и внешний цикл — wOuter = 1, wInner = 3;
    //  если EXIT не срабатывает — внутренний дойдёт до 10: wInner = 40.)
    wOuter := 0;
    wInner := 0;
    WHILE wOuter < 4 DO
        wOuter := wOuter + 1;
        wStep := 0;
        WHILE wStep < 10 DO
            wStep := wStep + 1;
            wInner := wInner + 1;
            IF wStep >= 3 THEN EXIT; END_IF;
        END_WHILE
    END_WHILE

    // ---- EXIT из REPEAT, вложенного в REPEAT ----

    // Внешний REPEAT — три итерации (UNTIL rOuter >= 3); внутренний на каждой
    // делает rStep 1, 2, 3 и выходит по EXIT. rOuter = 3, rInner = 3 * 3 = 9,
    // rStep = 3. (Если EXIT выбил бы и внешний — rOuter = 1, rInner = 3.)
    rOuter := 0;
    rInner := 0;
    REPEAT
        rOuter := rOuter + 1;
        rStep := 0;
        REPEAT
            rStep := rStep + 1;
            rInner := rInner + 1;
            IF rStep >= 3 THEN EXIT; END_IF;
        UNTIL rStep >= 9
        END_REPEAT
    UNTIL rOuter >= 3
    END_REPEAT

    // ---- EXIT в цикле внутри FUNCTION ----

    stepsUsed := StepsTo(20);           // 6 (без EXIT — 100)

    // ---- Условие-вызов функции в UNTIL ----

    // grown = 1, 2, 3 — Ripe ложно; на 4 истинно → выход. grown = 4.
    grown := 0;
    REPEAT
        grown := grown + 1;
    UNTIL Ripe(grown)
    END_REPEAT

    // ---- Условие-BOOL-переменная в WHILE ----

    // paces = 1, 2 — armed остаётся TRUE; paces = 3 → armed = FALSE, выход.
    // paces = 3, armed = 0. (Условие — переменная, а не сравнение: если бы
    //  оно вычислялось один раз до входа, как границы FOR, цикл не
    //  остановился бы вовсе — зависание, а не неверное число.)
    armed := TRUE;
    paces := 0;
    WHILE armed DO
        paces := paces + 1;
        armed := paces < 3;
    END_WHILE

    // ---- EXIT в REPEAT в теле ФБ, состояние между сканами ----

    // Скан 1: speed 0 → 3 → 6, на третьей итерации ticks = 2 → EXIT;
    //         rampSpeed = 6, rampTicks = 2, rampTop = 0 (6 >= 10 ложно).
    // Скан 2: 6 → 9 → 12, выход по UNTIL (12 >= 10); rampSpeed = 12,
    //         rampTicks = 2, rampTop = 1.
    // Скан 3: 12 → 15, выход по UNTIL сразу; rampSpeed = 15, rampTicks = 1,
    //         rampTop = 1.
    ramp(target := 10, speed => rampSpeed, ticks => rampTicks, atTop => rampTop);

    // ---- ELSIF по члену ФБ и по вызову функции ----

    // armed здесь всегда FALSE (его оставил цикл выше), поэтому решают ELSIF.
    // Скан 1: ramp.atTop ложно, Ripe(6) истинно → gear = 2.
    // Сканы 2 и 3: ramp.atTop истинно → gear = 1.
    // (Если ветки не связаны через else, то на сканах 2 и 3 сработает ещё и
    //  последняя ветка — Ripe(12) и Ripe(15) истинны: gear = 2 вместо 1.)
    IF armed THEN gear := 9;
    ELSIF ramp.atTop THEN gear := 1;
    ELSIF Ripe(rampSpeed) THEN gear := 2;
    ELSE gear := 3;
    END_IF;
END_PROGRAM
