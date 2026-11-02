#include <stdint.h>
#include <math.h>
#include <stdlib.h>
#include <stdio.h>

static float st2c_pow(float base, float exponent) {
    if (!isfinite(base) || !isfinite(exponent)) {
        fputs("**: range error (non-finite operand)\n", stderr);
        exit(EXIT_FAILURE);
    }
    if (exponent == 0.0f) return 1.0f;
    if ((base == 0.0f && exponent < 0.0f) ||
        (base < 0.0f && truncf(exponent) != exponent)) {
        fputs("**: domain error\n", stderr);
        exit(EXIT_FAILURE);
    }
    float result = powf(base, exponent);
    if (!isfinite(result)) {
        fputs("**: range error (overflow)\n", stderr);
        exit(EXIT_FAILURE);
    }
    return result;
}

typedef struct {
    int16_t expInt;
    float expReal;
    float chain;
    float signedBase;
    float outside;
    float inverse;
    float product;
    float root;
    float odd;
    float unit;
    float halfBase;
    float intSquare;
    float intRoot;
    float realSquare;
} OpsPower;
void OpsPower_init(OpsPower *self);
void OpsPower_step(OpsPower *self);

void OpsPower_init(OpsPower *self) {
    self->expInt = 3;
    self->expReal = 0.5f;
    self->chain = 0.0f;
    self->signedBase = 0.0f;
    self->outside = 0.0f;
    self->inverse = 0.0f;
    self->product = 0.0f;
    self->root = 0.0f;
    self->odd = 0.0f;
    self->unit = 0.0f;
    self->halfBase = 0.0f;
    self->intSquare = 0.0f;
    self->intRoot = 0.0f;
    self->realSquare = 0.0f;
}

void OpsPower_step(OpsPower *self) {
    float st2c_lhs, st2c_rhs;
    float st2c_lhs1, st2c_rhs1;
    float st2c_lhs2, st2c_rhs2;
    float st2c_lhs3, st2c_rhs3;
    float st2c_lhs4, st2c_rhs4;
    float st2c_lhs5, st2c_rhs5;
    float st2c_lhs6, st2c_rhs6;
    float st2c_lhs7, st2c_rhs7;
    float st2c_lhs8, st2c_rhs8;
    float st2c_lhs9, st2c_rhs9;
    float st2c_lhs10, st2c_rhs10;
    float st2c_lhs11, st2c_rhs11;
    float st2c_lhs12, st2c_rhs12;
    self->chain = (st2c_lhs1 = (st2c_lhs = 2.0f, st2c_rhs = 3, st2c_pow(st2c_lhs, st2c_rhs)), st2c_rhs1 = 2, st2c_pow(st2c_lhs1, st2c_rhs1));
    self->signedBase = (st2c_lhs2 = (-2.0f), st2c_rhs2 = 2, st2c_pow(st2c_lhs2, st2c_rhs2));
    self->outside = (-(st2c_lhs3 = 2.0f, st2c_rhs3 = 2, st2c_pow(st2c_lhs3, st2c_rhs3)));
    self->inverse = (st2c_lhs4 = 2.0f, st2c_rhs4 = (-2), st2c_pow(st2c_lhs4, st2c_rhs4));
    self->product = (2.0f * (st2c_lhs5 = 3.0f, st2c_rhs5 = 2, st2c_pow(st2c_lhs5, st2c_rhs5)));
    self->root = (st2c_lhs6 = 9.0f, st2c_rhs6 = self->expReal, st2c_pow(st2c_lhs6, st2c_rhs6));
    self->odd = (st2c_lhs7 = (-2.0f), st2c_rhs7 = self->expInt, st2c_pow(st2c_lhs7, st2c_rhs7));
    self->unit = (st2c_lhs8 = 0.0f, st2c_rhs8 = 0, st2c_pow(st2c_lhs8, st2c_rhs8));
    self->halfBase = (st2c_lhs9 = (1.0f / 2.0f), st2c_rhs9 = 2, st2c_pow(st2c_lhs9, st2c_rhs9));
    self->intSquare = (st2c_lhs10 = (float)self->expInt, st2c_rhs10 = 2, st2c_pow(st2c_lhs10, st2c_rhs10));
    self->intRoot = (st2c_lhs11 = (float)(self->expInt + 1), st2c_rhs11 = self->expReal, st2c_pow(st2c_lhs11, st2c_rhs11));
    self->realSquare = (st2c_lhs12 = self->expReal, st2c_rhs12 = 2, st2c_pow(st2c_lhs12, st2c_rhs12));
}

int main(void) {
    OpsPower st;
    OpsPower_init(&st);
    for (int scan = 0; scan < 1; ++scan) {
        OpsPower_step(&st);
        printf("expInt=%d\n", st.expInt);
        printf("expReal=%g\n", st.expReal);
        printf("chain=%g\n", st.chain);
        printf("signedBase=%g\n", st.signedBase);
        printf("outside=%g\n", st.outside);
        printf("inverse=%g\n", st.inverse);
        printf("product=%g\n", st.product);
        printf("root=%g\n", st.root);
        printf("odd=%g\n", st.odd);
        printf("unit=%g\n", st.unit);
        printf("halfBase=%g\n", st.halfBase);
        printf("intSquare=%g\n", st.intSquare);
        printf("intRoot=%g\n", st.intRoot);
        printf("realSquare=%g\n", st.realSquare);
    }
    return 0;
}
