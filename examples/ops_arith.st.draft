(* Один скан. Остаток после деления с усечением к нулю:
   pp=1, np=-1, pn=1, nn=-1. В цепочке 20 MOD 6 * 3 = 2 * 3,
   поэтому chain=6 (ошибка приоритета дала бы 2).
   small=2, exact=0, edge=0, zero=0, dynamicZero=0.
   signs=-32768 (плюс сохраняет минус), adapted=0.5 (оба литерала REAL). *)
PROGRAM OpsArith
VAR
    pp, np, pn, nn, chain, small, exact, edge, zero, dynamicZero, signs : INT;
    adapted : REAL;
END_VAR
pp := 7 MOD 3;
np := -7 MOD 3;
pn := 7 MOD -3;
nn := -7 MOD -3;
chain := 20 MOD 6 * 3;
small := +2 MOD 5;
exact := 6 MOD 3;
edge := -32768 MOD -1;
zero := 7 MOD 0;
dynamicZero := 9 MOD zero;
signs := +(-32768);
adapted := +1 / 2;
END_PROGRAM
