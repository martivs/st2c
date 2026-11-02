(* Три скана. Pick(1) возвращает нулевой результат, Pick(2/3) — 4/6.
   В ФБ count каждый раз растёт на 1, ещё на 10 только во втором вызове.
   RETURN из ФБ сохраняет выгрузку =>, из PROGRAM пропускает хвост только
   второго скана. Поля: scan=1/2/3, picked=0/4/6, seen=1/12/13,
   tail=1/1/2. Сброс состояния или завершение всего процесса это нарушит. *)
FUNCTION Pick : INT
VAR_INPUT n : INT; END_VAR
IF n = 1 THEN RETURN; END_IF;
Pick := n * 2;
RETURN;
Pick := 99;
END_FUNCTION

FUNCTION_BLOCK Meter
VAR_INPUT shortStep : BOOL; END_VAR
VAR_OUTPUT count : INT; END_VAR
count := count + 1;
IF shortStep THEN RETURN; END_IF;
count := count + 10;
END_FUNCTION_BLOCK

PROGRAM OpsReturn
VAR
    meterInst : Meter;
    scan, picked, seen, tail : INT;
END_VAR
scan := scan + 1;
picked := Pick(scan);
meterInst(shortStep := scan <> 2, count => seen);
IF scan = 2 THEN RETURN; END_IF;
tail := tail + 1;
END_PROGRAM
