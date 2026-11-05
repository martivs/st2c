(* Четыре скана: FUNCTION + CASE/MOD/** + два независимых Channel,
   вложенный Bucket, частичная передача входов, RETURN из ФБ и => во вход
   другого ФБ. PROGRAM намеренно раньше определений используемых POU.

   Sample: n=1 -> (1*0.5)^2=0.25; n=2 -> 2*0.5=1;
   n=3 -> ранний возврат с нулём; n=4 -> (4*0.5)^2=4.
   Левый канал включён на сканах 1/2/4, правый только на скане 3.
   Пропущенный вход step правого канала сохраняет 0.5 со скана 1.
   Sink на каждом скане прибавляет полный текущий total левого канала,
   в том числе при его раннем RETURN на скане 3: выгрузка => обязательна.
   Перед вызовом sink.delta получает -100: пропущенная выгрузка заметна,
   даже если сам total не изменился относительно предыдущего скана.

   Поле       | Скан 1 | Скан 2 | Скан 3 | Скан 4
   scan       | 1      | 2      | 3      | 4
   increment  | 0.25   | 1      | 0      | 4
   leftTotal  | 0.25   | 1.25   | 1.25   | 5.25
   rightTotal | 0      | 0      | 0.5    | 0.5
   combined   | 0.25   | 1.5    | 3.25   | 8.5
   enabled    | 1      | 1      | 0      | 1
   ready      | 0      | 1      | 0      | 1
   Все REAL-значения точны в binary32. *)
PROGRAM MixedPipeline
VAR
    scan : INT;
    increment, leftTotal, rightTotal, combined : REAL;
    enabled, ready : BOOL;
    left, right : Channel;
    sink : Bucket;
END_VAR
scan := scan + 1;
enabled := +scan MOD 3 <> 0;
increment := Sample(gain := 1 / 2, n := scan);
IF scan = 1 THEN
    right.step := 1 / 2;
END_IF;
sink.delta := -100;
left(step := increment, enable := enabled, total => sink.delta);
leftTotal := left.total;
sink();
right(enable := NOT enabled, total => rightTotal);
combined := sink.total + rightTotal;
ready := left.ready XOR right.ready;
END_PROGRAM

FUNCTION_BLOCK Channel
VAR_INPUT
    step : REAL;
    enable : BOOL := TRUE;
END_VAR
VAR_OUTPUT
    total : REAL;
    ready : BOOL;
END_VAR
VAR
    inner : Bucket;
END_VAR
IF NOT enable THEN
    ready := FALSE;
    RETURN;
END_IF;
inner(delta := step, total => total);
ready := total >= 1;
END_FUNCTION_BLOCK

FUNCTION Sample : REAL
VAR_INPUT
    n : INT;
    gain : REAL;
END_VAR
CASE n MOD 3 OF
    0: RETURN;
    1: Sample := (INT_TO_REAL(n) * gain) ** 2;
       RETURN;
    2: Sample := INT_TO_REAL(n) * gain;
END_CASE;
END_FUNCTION

FUNCTION_BLOCK Bucket
VAR_INPUT delta : REAL; END_VAR
VAR_OUTPUT total : REAL; END_VAR
total := total + delta;
END_FUNCTION_BLOCK
