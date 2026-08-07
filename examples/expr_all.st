(* Выражения: все шесть сравнений, унарный минус, скобки, смешанные
   приоритеты и левая ассоциативность.

   Проверяет генерацию выражений со скобками вокруг каждой бинарной операции.
   Итог cmp — битовая маска: каждое сравнение зажигает свой бит, поэтому по
   значению видно, какое именно сломалось. Все шесть верны → cmp = 127. *)
PROGRAM ExprAll
VAR
    a : INT := 20;
    b : INT := 6;
    c : INT := 3;
    assoc : INT;
    prio : INT;
    unary : INT;
    cmp : INT;
END_VAR

assoc := a - b - c;         // левая ассоциативность: (20 - 6) - 3 = 11
prio  := a - b * c;         // приоритет умножения: 20 - (6 * 3) = 2
unary := -a + b * c - 2;    // ((-20) + 18) - 2 = -4

cmp := 0;
IF a > b THEN cmp := cmp + 1; END_IF
IF b < a THEN cmp := cmp + 2; END_IF
IF a >= 20 THEN cmp := cmp + 4; END_IF
IF b <= 6 THEN cmp := cmp + 8; END_IF
IF c = 3 THEN cmp := cmp + 16; END_IF
IF c <> 4 THEN cmp := cmp + 32; END_IF
IF (a - b) * c = 42 THEN cmp := cmp + 64; END_IF   // скобки против приоритета

END_PROGRAM
