#include <stdint.h>
#include <stdio.h>

typedef struct {
    int16_t a;
    int16_t b;
    int16_t c;
    int16_t assoc;
    int16_t prio;
    int16_t unary;
    int16_t cmp;
} ExprAll;
void ExprAll_init(ExprAll *self);
void ExprAll_step(ExprAll *self);

void ExprAll_init(ExprAll *self) {
    self->a = 20;
    self->b = 6;
    self->c = 3;
    self->assoc = 0;
    self->prio = 0;
    self->unary = 0;
    self->cmp = 0;
}

void ExprAll_step(ExprAll *self) {
    self->assoc = ((self->a - self->b) - self->c);
    self->prio = (self->a - (self->b * self->c));
    self->unary = (((-self->a) + (self->b * self->c)) - 2);
    self->cmp = 0;
    if ((self->a > self->b)) {
        self->cmp = (self->cmp + 1);
    }
    if ((self->b < self->a)) {
        self->cmp = (self->cmp + 2);
    }
    if ((self->a >= 20)) {
        self->cmp = (self->cmp + 4);
    }
    if ((self->b <= 6)) {
        self->cmp = (self->cmp + 8);
    }
    if ((self->c == 3)) {
        self->cmp = (self->cmp + 16);
    }
    if ((self->c != 4)) {
        self->cmp = (self->cmp + 32);
    }
    if ((((self->a - self->b) * self->c) == 42)) {
        self->cmp = (self->cmp + 64);
    }
}

int main(void) {
    ExprAll st;
    ExprAll_init(&st);
    for (int scan = 0; scan < 1; ++scan) {
        ExprAll_step(&st);
        printf("a=%d\n", st.a);
        printf("b=%d\n", st.b);
        printf("c=%d\n", st.c);
        printf("assoc=%d\n", st.assoc);
        printf("prio=%d\n", st.prio);
        printf("unary=%d\n", st.unary);
        printf("cmp=%d\n", st.cmp);
    }
    return 0;
}
