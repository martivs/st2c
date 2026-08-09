#include <stdint.h>
#include <stdio.h>

int16_t Add(int16_t x, int16_t y);

int16_t Clamp(int16_t v, int16_t lo, int16_t hi);

typedef struct {
    int16_t plain;
    int16_t nested;
    int16_t named;
    int16_t clamped;
} FuncSimple;
void FuncSimple_init(FuncSimple *self);
void FuncSimple_step(FuncSimple *self);

int16_t Add(int16_t x, int16_t y) {
    int16_t Add = 0;
    Add = (x + y);
    return Add;
}

int16_t Clamp(int16_t v, int16_t lo, int16_t hi) {
    int16_t Clamp = 0;
    Clamp = v;
    if ((v < lo)) {
        Clamp = lo;
    }
    if ((v > hi)) {
        Clamp = hi;
    }
    return Clamp;
}

void FuncSimple_init(FuncSimple *self) {
    self->plain = 0;
    self->nested = 0;
    self->named = 0;
    self->clamped = 0;
}

void FuncSimple_step(FuncSimple *self) {
    self->plain = Add(2, 3);
    self->nested = Add(Add(1, 2), Add(3, 4));
    self->named = Add(10, 5);
    self->clamped = Clamp(99, 0, 50);
}

int main(void) {
    FuncSimple st;
    FuncSimple_init(&st);
    for (int scan = 0; scan < 1; ++scan) {
        FuncSimple_step(&st);
        printf("plain=%d\n", st.plain);
        printf("nested=%d\n", st.nested);
        printf("named=%d\n", st.named);
        printf("clamped=%d\n", st.clamped);
    }
    return 0;
}
