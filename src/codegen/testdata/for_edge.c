#include <stdint.h>
#include <stdio.h>

typedef struct {
    int16_t i;
    int16_t j;
    int16_t up;
    int16_t down;
    int16_t nested;
    int16_t limit;
    int16_t edge;
} ForEdge;
void ForEdge_init(ForEdge *self);
void ForEdge_step(ForEdge *self);

void ForEdge_init(ForEdge *self) {
    self->i = 0;
    self->j = 0;
    self->up = 0;
    self->down = 0;
    self->nested = 0;
    self->limit = 4;
    self->edge = 0;
}

void ForEdge_step(ForEdge *self) {
    self->up = 0;
    {
        int32_t __i0 = 1, __end0 = 9, __step0 = 2;
        for (; __i0 <= __end0; __i0 += __step0) {
            self->i = (int16_t)__i0;
            self->up = (self->up + self->i);
        }
    }
    self->down = 0;
    {
        int32_t __i1 = 10, __end1 = 1, __step1 = (-3);
        for (; __i1 >= __end1; __i1 += __step1) {
            self->i = (int16_t)__i1;
            self->down = (self->down + self->i);
        }
    }
    self->nested = 0;
    {
        int32_t __i2 = 1, __end2 = self->limit, __step2 = 1;
        for (; __i2 <= __end2; __i2 += __step2) {
            self->i = (int16_t)__i2;
            {
                int32_t __i3 = 1, __end3 = self->i, __step3 = 1;
                for (; __i3 <= __end3; __i3 += __step3) {
                    self->j = (int16_t)__i3;
                    self->nested = (self->nested + 1);
                }
            }
        }
    }
    self->edge = 0;
    {
        int32_t __i4 = 32760, __end4 = 32767, __step4 = 1;
        for (; __i4 <= __end4; __i4 += __step4) {
            self->i = (int16_t)__i4;
            self->edge = (self->edge + 1);
        }
    }
}

int main(void) {
    ForEdge st;
    ForEdge_init(&st);
    for (int scan = 0; scan < 1; ++scan) {
        ForEdge_step(&st);
        printf("i=%d\n", st.i);
        printf("j=%d\n", st.j);
        printf("up=%d\n", st.up);
        printf("down=%d\n", st.down);
        printf("nested=%d\n", st.nested);
        printf("limit=%d\n", st.limit);
        printf("edge=%d\n", st.edge);
    }
    return 0;
}
