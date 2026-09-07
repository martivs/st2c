(* Тип REAL: объявления с инициализаторами (дробь, знак, экспонента),
   арифметика и вещественное деление, адаптивный целый литерал, сравнения
   в IF, конверсии INT_TO_REAL / REAL_TO_INT с округлением к ближайшему,
   REAL как параметр и возврат FUNCTION, как вход и выход FUNCTION_BLOCK,
   целочисленный FOR с накоплением в REAL.

   Все величины точны в двоичном представлении (0.5, 0.25, 2.5, …): драйвер
   печатает REAL через %g (6 значащих цифр), и эталон не должен зависеть от
   округления float.

   Ожидание (один скан): a=2.5 b=0.25 neg=-1.5 big=1500 zero=0 quot=0.5
   half=1.75 scaled=0.75 conv=2.75 rounded=3 negRounded=-2 cmp=7 i=4 acc=5
   total=0.6875. Расчёт — в комментариях у операторов. *)

FUNCTION Half : REAL
VAR_INPUT
    x : REAL;
END_VAR
    Half := x / 2;                      // литерал 2 адаптируется к REAL: 2.0f
END_FUNCTION

FUNCTION Scale : REAL
VAR_INPUT
    v : REAL;
    k : INT;
END_VAR
    Scale := v * INT_TO_REAL(k);
END_FUNCTION

FUNCTION_BLOCK Integrator
VAR_INPUT
    v : REAL;
END_VAR
VAR_OUTPUT
    total : REAL;
END_VAR
VAR
    dt : REAL := 0.25;
END_VAR
    total := total + v * dt;
END_FUNCTION_BLOCK

PROGRAM RealAll
VAR
    a : REAL := 2.5;
    b : REAL := 0.25;
    neg : REAL := -1.5;
    big : REAL := 1.5E3;                // экспонента: 1500
    zero : REAL;                        // без инициализатора: 0.0f
    quot : REAL;
    half : REAL;
    scaled : REAL;
    conv : REAL;
    rounded : INT;
    negRounded : INT;
    cmp : INT;
    i : INT;
    acc : REAL;
    total : REAL;
    integ : Integrator;
END_VAR

quot := 1 / 2;                          // адаптивный литерал: (1.0f / 2.0f) = 0.5
half := Half(a + 1);                    // (2.5 + 1.0) / 2 = 1.75
scaled := Scale(b, 3);                  // 0.25 * 3 = 0.75
rounded := REAL_TO_INT(a);              // 2.5 → 3 (к ближайшему, от нуля)
negRounded := REAL_TO_INT(neg);         // -1.5 → -2
conv := INT_TO_REAL(rounded) - b;       // 3 - 0.25 = 2.75

cmp := 0;
IF a > b THEN cmp := cmp + 1; END_IF          // 2.5 > 0.25 — да: 1
IF a * b = 0.625 THEN cmp := cmp + 2; END_IF  // 2.5 * 0.25 = 0.625 точно: 3
IF zero < 1 THEN cmp := cmp + 4; END_IF       // литерал 1 → 1.0f; 0 < 1: 7
IF neg >= 0 THEN cmp := cmp + 8; END_IF       // -1.5 >= 0 — нет: 7
IF b <> 0.25 THEN cmp := cmp + 16; END_IF     // равны — нет: 7

acc := 0;                               // литерал 0 → 0.0f
FOR i := 1 TO 4 DO
    acc := acc + INT_TO_REAL(i) * 0.5;  // 0.5 + 1 + 1.5 + 2 = 5; i = 4 после цикла
END_FOR

integ(v := a);                          // total = 0 + 2.5 * 0.25 = 0.625
integ(v := b, total => total);          // total = 0.625 + 0.25 * 0.25 = 0.6875

END_PROGRAM
