#include <stdint.h>
#include <stdio.h>

typedef struct {
    int16_t i;
    int16_t j;
    int16_t acc;
} DeeplyNested;
void DeeplyNested_init(DeeplyNested *self);
void DeeplyNested_step(DeeplyNested *self);

void DeeplyNested_init(DeeplyNested *self) {
    self->i = 0;
    self->j = 0;
    self->acc = 0;
}

void DeeplyNested_step(DeeplyNested *self) {
    self->acc = 0;
    {
        int32_t __i0 = 1, __end0 = 3, __step0 = 1;
        for (; __i0 <= __end0; __i0 += __step0) {
            self->i = (int16_t)__i0;
            {
                int32_t __i1 = 1, __end1 = 3, __step1 = 1;
                for (; __i1 <= __end1; __i1 += __step1) {
                    self->j = (int16_t)__i1;
                    if (((self->i + self->j) > 4)) {
                        if ((self->i == self->j)) {
                            self->acc = (self->acc + 1);
                        } else {
                            self->acc = (self->acc + 2);
                        }
                    } else {
                        self->acc = (self->acc - 1);
                    }
                }
            }
        }
    }
}

int main(void) {
    DeeplyNested st;
    DeeplyNested_init(&st);
    for (int scan = 0; scan < 1; ++scan) {
        DeeplyNested_step(&st);
        printf("i=%d\n", st.i);
        printf("j=%d\n", st.j);
        printf("acc=%d\n", st.acc);
    }
    return 0;
}
