(* Функциональный блок: состояние, переживающее вызов, два независимых
   экземпляра, входы через :=, выходы через =>, чтение члена в выражении.

   Два экземпляра здесь принципиальны: на одном экземпляре ошибка «состояние
   утекло в глобальную переменную вместо поля структуры» не видна вовсе.
   fast и slow обязаны считать независимо.

   Смысл проявляется только за несколько сканов, поэтому проверять так:
       go run ./src -main -scans 3 -o /tmp/a.c examples/fb_counter.st
   Ожидание по сканам: fastOut = 10, 20, 30; slow.count = 1, 2, 3;
   total = 11, 22, 33.

   ВНИМАНИЕ: разбирается начиная с этапа 3, исполняется с этапа 7. *)

FUNCTION_BLOCK Counter
VAR_INPUT
    step : INT;
END_VAR
VAR_OUTPUT
    count : INT;
END_VAR
VAR
    calls : INT;
END_VAR
    calls := calls + 1;
    count := count + step;
END_FUNCTION_BLOCK

PROGRAM FbCounter
VAR
    fast : Counter;
    slow : Counter;
    fastOut : INT;
    total : INT;
END_VAR

fast(step := 10, count => fastOut);
slow(step := 1);                      // вызов без привязки выхода — законен
total := fastOut + slow.count;

END_PROGRAM
