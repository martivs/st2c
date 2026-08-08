#include <stdint.h>
#include <stdio.h>

typedef struct {
    int16_t x;
    int16_t y;
    int16_t sum;
} Example;
void Example_init(Example *self);
void Example_step(Example *self);

void Example_init(Example *self) {
    self->x = 0;
    self->y = 0;
    self->sum = 0;
}

void Example_step(Example *self) {
    self->x = 10;
    self->y = 3;
    self->sum = 0;
    {
        int32_t __i0 = 1, __end0 = 5, __step0 = 1;
        for (; __i0 <= __end0; __i0 += __step0) {
            self->x = (int16_t)__i0;
            self->sum = (self->sum + self->x);
        }
    }
    if ((self->sum > 10)) {
        self->y = 1;
    } else {
        self->y = 0;
    }
}

int main(void) {
    Example st;
    Example_init(&st);
    for (int scan = 0; scan < 1; ++scan) {
        Example_step(&st);
        printf("x=%d\n", st.x);
        printf("y=%d\n", st.y);
        printf("sum=%d\n", st.sum);
    }
    return 0;
}
