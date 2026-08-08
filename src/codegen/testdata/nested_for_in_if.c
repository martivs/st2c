#include <stdint.h>
#include <stdio.h>

typedef struct {
    int16_t x;
    int16_t i;
    int16_t total;
} NestedForInIf;
void NestedForInIf_init(NestedForInIf *self);
void NestedForInIf_step(NestedForInIf *self);

void NestedForInIf_init(NestedForInIf *self) {
    self->x = 0;
    self->i = 0;
    self->total = 0;
}

void NestedForInIf_step(NestedForInIf *self) {
    self->x = 7;
    self->total = 0;
    if ((self->x > 5)) {
        {
            int32_t __i0 = 1, __end0 = self->x, __step0 = 1;
            for (; __i0 <= __end0; __i0 += __step0) {
                self->i = (int16_t)__i0;
                self->total = (self->total + self->i);
            }
        }
    } else {
        self->total = 0;
    }
}

int main(void) {
    NestedForInIf st;
    NestedForInIf_init(&st);
    for (int scan = 0; scan < 1; ++scan) {
        NestedForInIf_step(&st);
        printf("x=%d\n", st.x);
        printf("i=%d\n", st.i);
        printf("total=%d\n", st.total);
    }
    return 0;
}
