#include <stdint.h>
#include <stdio.h>

typedef struct {
    int16_t scan;
    int16_t band;
    int16_t hits;
    int16_t ticks;
    int16_t tails;
} OpsCase;
void OpsCase_init(OpsCase *self);
void OpsCase_step(OpsCase *self);

void OpsCase_init(OpsCase *self) {
    self->scan = 0;
    self->band = 0;
    self->hits = 0;
    self->ticks = 0;
    self->tails = 0;
}

void OpsCase_step(OpsCase *self) {
    self->scan = (self->scan + 1);
    {
        int32_t st2c_sel = (self->scan - 2);
        int st2c_arm = 0;
        if ((st2c_sel >= -2 && st2c_sel <= -1) || st2c_sel == 1) st2c_arm = 1;
        switch (st2c_arm) {
        case 1:
        {
            self->band = 7;
            break;
        }
        default:
        {
            self->band = 8;
            break;
        }
        }
    }
    self->hits = 0;
    self->ticks = 0;
    self->tails = 0;
    {
        int st2c_exit = 0;
        while ((self->hits < 3)) {
            self->hits = (self->hits + 1);
            {
                int32_t st2c_sel1 = self->hits;
                switch (st2c_sel1) {
                case 1:
                {
                    do {
                        self->ticks = (self->ticks + 1);
                        break;
                    } while (!(1));
                    self->ticks = (self->ticks + 10);
                    break;
                }
                case 2:
                {
                    {
                        int32_t st2c_sel2 = self->band;
                        switch (st2c_sel2) {
                        case 7:
                        case 8:
                        {
                            st2c_exit = 1;
                            break;
                            break;
                        }
                        }
                        if (st2c_exit) break;
                    }
                    self->ticks = (self->ticks + 100);
                    break;
                }
                }
                if (st2c_exit) break;
            }
            self->tails = (self->tails + 1);
        }
    }
}

int main(void) {
    OpsCase st;
    OpsCase_init(&st);
    for (int scan = 0; scan < 3; ++scan) {
        OpsCase_step(&st);
        printf("scan=%d\n", st.scan);
        printf("band=%d\n", st.band);
        printf("hits=%d\n", st.hits);
        printf("ticks=%d\n", st.ticks);
        printf("tails=%d\n", st.tails);
    }
    return 0;
}
