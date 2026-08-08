#include <stdint.h>
#include <stdio.h>

typedef struct {
    int16_t i;
    int16_t sum;
    int16_t flag;
} NestedIfInFor;
void NestedIfInFor_init(NestedIfInFor *self);
void NestedIfInFor_step(NestedIfInFor *self);

void NestedIfInFor_init(NestedIfInFor *self) {
    self->i = 0;
    self->sum = 0;
    self->flag = 0;
}

void NestedIfInFor_step(NestedIfInFor *self) {
    self->sum = 0;
    self->flag = 0;
    {
        int32_t __i0 = 1, __end0 = 10, __step0 = 1;
        for (; __i0 <= __end0; __i0 += __step0) {
            self->i = (int16_t)__i0;
            if ((self->i > 5)) {
                self->sum = (self->sum + self->i);
            } else {
                self->flag = (self->flag + 1);
            }
        }
    }
}

int main(void) {
    NestedIfInFor st;
    NestedIfInFor_init(&st);
    for (int scan = 0; scan < 1; ++scan) {
        NestedIfInFor_step(&st);
        printf("i=%d\n", st.i);
        printf("sum=%d\n", st.sum);
        printf("flag=%d\n", st.flag);
    }
    return 0;
}
