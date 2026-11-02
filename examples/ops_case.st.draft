(* Три скана: scan=1/2/3. Селектор scan-2 даёт -1/0/1:
   для списка (-2..-1, +1) band=7, иначе 8, итого band=7/8/7.
   На первом обороте WHILE внутренний REPEAT завершается своим EXIT,
   после него ticks растёт ещё на 10. На втором обороте EXIT из двух
   CASE сразу покидает WHILE. hits=2/2/2, ticks=11/11/11, tails=1/1/1.
   Проваливание, потерянный EXIT и выход не из того цикла различимы. *)
PROGRAM OpsCase
VAR scan, band, hits, ticks, tails : INT; END_VAR
scan := scan + 1;
CASE scan - 2 OF
    -2..-1, +1: band := 7;
    ELSE band := 8;
END_CASE;
hits := 0;
ticks := 0;
tails := 0;
WHILE hits < 3 DO
    hits := hits + 1;
    CASE hits OF
        1: REPEAT
               ticks := ticks + 1;
               EXIT;
           UNTIL TRUE END_REPEAT;
           ticks := ticks + 10;
        2: CASE band OF
               7, 8: EXIT;
           END_CASE;
           ticks := ticks + 100;
    END_CASE;
    tails := tails + 1;
END_WHILE;
END_PROGRAM
