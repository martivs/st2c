#include <stdint.h>
#include <stdio.h>

static int16_t st2c_mod(int16_t lhs, int16_t rhs) {
    return rhs == 0 ? 0 : (int32_t)lhs % (int32_t)rhs;
}

typedef struct {
    int16_t pp;
    int16_t np;
    int16_t pn;
    int16_t nn;
    int16_t chain;
    int16_t small;
    int16_t exact;
    int16_t edge;
    int16_t zero;
    int16_t dynamicZero;
    int16_t signs;
    float adapted;
} OpsArith;
void OpsArith_init(OpsArith *self);
void OpsArith_step(OpsArith *self);

void OpsArith_init(OpsArith *self) {
    self->pp = 0;
    self->np = 0;
    self->pn = 0;
    self->nn = 0;
    self->chain = 0;
    self->small = 0;
    self->exact = 0;
    self->edge = 0;
    self->zero = 0;
    self->dynamicZero = 0;
    self->signs = 0;
    self->adapted = 0.0f;
}

void OpsArith_step(OpsArith *self) {
    int16_t st2c_lhs, st2c_rhs;
    int16_t st2c_lhs1, st2c_rhs1;
    int16_t st2c_lhs2, st2c_rhs2;
    int16_t st2c_lhs3, st2c_rhs3;
    int16_t st2c_lhs4, st2c_rhs4;
    int16_t st2c_lhs5, st2c_rhs5;
    int16_t st2c_lhs6, st2c_rhs6;
    int16_t st2c_lhs7, st2c_rhs7;
    int16_t st2c_lhs8, st2c_rhs8;
    int16_t st2c_lhs9, st2c_rhs9;
    self->pp = (st2c_lhs = 7, st2c_rhs = 3, st2c_mod(st2c_lhs, st2c_rhs));
    self->np = (st2c_lhs1 = (-7), st2c_rhs1 = 3, st2c_mod(st2c_lhs1, st2c_rhs1));
    self->pn = (st2c_lhs2 = 7, st2c_rhs2 = (-3), st2c_mod(st2c_lhs2, st2c_rhs2));
    self->nn = (st2c_lhs3 = (-7), st2c_rhs3 = (-3), st2c_mod(st2c_lhs3, st2c_rhs3));
    self->chain = ((st2c_lhs4 = 20, st2c_rhs4 = 6, st2c_mod(st2c_lhs4, st2c_rhs4)) * 3);
    self->small = (st2c_lhs5 = (+2), st2c_rhs5 = 5, st2c_mod(st2c_lhs5, st2c_rhs5));
    self->exact = (st2c_lhs6 = 6, st2c_rhs6 = 3, st2c_mod(st2c_lhs6, st2c_rhs6));
    self->edge = (st2c_lhs7 = (-32768), st2c_rhs7 = (-1), st2c_mod(st2c_lhs7, st2c_rhs7));
    self->zero = (st2c_lhs8 = 7, st2c_rhs8 = 0, st2c_mod(st2c_lhs8, st2c_rhs8));
    self->dynamicZero = (st2c_lhs9 = 9, st2c_rhs9 = self->zero, st2c_mod(st2c_lhs9, st2c_rhs9));
    self->signs = (+(-32768));
    self->adapted = ((+1.0f) / 2.0f);
}

int main(void) {
    OpsArith st;
    OpsArith_init(&st);
    for (int scan = 0; scan < 1; ++scan) {
        OpsArith_step(&st);
        printf("pp=%d\n", st.pp);
        printf("np=%d\n", st.np);
        printf("pn=%d\n", st.pn);
        printf("nn=%d\n", st.nn);
        printf("chain=%d\n", st.chain);
        printf("small=%d\n", st.small);
        printf("exact=%d\n", st.exact);
        printf("edge=%d\n", st.edge);
        printf("zero=%d\n", st.zero);
        printf("dynamicZero=%d\n", st.dynamicZero);
        printf("signs=%d\n", st.signs);
        printf("adapted=%g\n", st.adapted);
    }
    return 0;
}
