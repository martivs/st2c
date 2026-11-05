#include <stdint.h>
#include <math.h>
#include <stdlib.h>
#include <stdio.h>

static int16_t st_real_to_int(float v) {
    return (int16_t)(v >= 0.0f ? v + 0.5f : v - 0.5f);
}

static int16_t st2c_mod(int16_t lhs, int16_t rhs) {
    return rhs == 0 ? 0 : (int32_t)lhs % (int32_t)rhs;
}

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

float Blend(float x, int16_t n, _Bool gate);

typedef struct {
    float seed;
    int16_t n;
    int16_t negative;
    float unit;
    float a;
    float b;
    float c;
    float nested;
    float st_powf;
    int16_t rounded;
    int16_t ticks;
    int16_t i;
    int16_t st2c_lhs;
    _Bool safe;
    _Bool logic;
} MixedNumeric;
void MixedNumeric_init(MixedNumeric *self);
void MixedNumeric_step(MixedNumeric *self);

float Blend(float x, int16_t n, _Bool gate) {
    float st2c_lhs1, st2c_rhs;
    int16_t st2c_lhs2, st2c_rhs1;
    int16_t st2c_lhs3, st2c_rhs2;
    float Blend = 0.0f;
    float base = 0.0f;
    int16_t st_remainder = 0;
    float half = 0.0f;
    base = (st2c_lhs1 = x, st2c_rhs = 2, st2c_pow(st2c_lhs1, st2c_rhs));
    st_remainder = (st2c_lhs3 = (st2c_lhs2 = (+n), st2c_rhs1 = 5, st2c_mod(st2c_lhs2, st2c_rhs1)), st2c_rhs2 = 3, st2c_mod(st2c_lhs3, st2c_rhs2));
    half = (1.0f / 2.0f);
    if ((!gate)) {
        return Blend;
    }
    {
        int st2c_exit = 0;
        do {
            {
                int32_t st2c_sel = st_remainder;
                int st2c_arm = 0;
                if (st2c_sel == 0) st2c_arm = 1;
                else if ((st2c_sel >= 1 && st2c_sel <= 2)) st2c_arm = 2;
                switch (st2c_arm) {
                case 1:
                {
                    Blend = (base + half);
                    return Blend;
                    break;
                }
                case 2:
                {
                    Blend = (base / (((float)(st_remainder)) + 1.0f));
                    st2c_exit = 1;
                    break;
                    break;
                }
                default:
                {
                    Blend = (-1.0f);
                    break;
                }
                }
                if (st2c_exit) break;
            }
        } while (!(1));
    }
    return Blend;
}

void MixedNumeric_init(MixedNumeric *self) {
    float st2c_lhs4, st2c_rhs3;
    int16_t st2c_lhs5, st2c_rhs4;
    int16_t st2c_lhs6, st2c_rhs5;
    float st2c_lhs7, st2c_rhs6;
    self->seed = (st2c_lhs4 = (1.0f / 2.0f), st2c_rhs3 = 2, st2c_pow(st2c_lhs4, st2c_rhs3));
    self->n = (st2c_lhs5 = 16, st2c_rhs4 = 5, st2c_mod(st2c_lhs5, st2c_rhs4));
    self->negative = (st2c_lhs6 = (-17), st2c_rhs5 = 5, st2c_mod(st2c_lhs6, st2c_rhs5));
    self->unit = (st2c_lhs7 = 2.0f, st2c_rhs6 = (1 / 2), st2c_pow(st2c_lhs7, st2c_rhs6));
    self->a = 0.0f;
    self->b = 0.0f;
    self->c = 0.0f;
    self->nested = 0.0f;
    self->st_powf = 0.0f;
    self->rounded = 0;
    self->ticks = 0;
    self->i = 0;
    self->st2c_lhs = 0;
    self->safe = 0;
    self->logic = 0;
}

void MixedNumeric_step(MixedNumeric *self) {
    float st2c_lhs8, st2c_rhs7;
    int16_t st2c_lhs9, st2c_rhs8;
    int16_t st2c_lhs10, st2c_rhs9;
    float st2c_lhs11, st2c_rhs10;
    float st2c_lhs12, st2c_rhs11;
    self->a = Blend((self->seed + (1.0f / 4.0f)), self->n, 1);
    self->b = Blend(2.0f, 5, 1);
    self->c = Blend((1.0f / 2.0f), 7, 0);
    self->nested = Blend(Blend(1.0f, 5, 1), 6, 1);
    self->st_powf = ((self->a + self->b) + self->nested);
    self->rounded = st_real_to_int(self->st_powf);
    while (((st2c_lhs8 = 2.0f, st2c_rhs7 = self->ticks, st2c_pow(st2c_lhs8, st2c_rhs7)) < 8.0f)) {
        self->ticks = (self->ticks + 1);
    }
    do {
        self->n = (self->n + 1);
    } while (!(((st2c_lhs9 = self->n, st2c_rhs8 = 4, st2c_mod(st2c_lhs9, st2c_rhs8)) == 0)));
    {
        int32_t __i0 = 3, __end0 = 1, __step0 = (-(1 + (st2c_lhs10 = self->n, st2c_rhs9 = 2, st2c_mod(st2c_lhs10, st2c_rhs9))));
        for (; __step0 >= 0 ? __i0 <= __end0 : __i0 >= __end0; __i0 += __step0) {
            self->i = (int16_t)__i0;
            self->st2c_lhs = (self->st2c_lhs + self->i);
        }
    }
    self->safe = (((self->n == 4) || ((st2c_lhs11 = 0.0f, st2c_rhs10 = (-1), st2c_pow(st2c_lhs11, st2c_rhs10)) > 0.0f)) && (!((self->n != 4) && ((st2c_lhs12 = (-2.0f), st2c_rhs11 = 0.5f, st2c_pow(st2c_lhs12, st2c_rhs11)) > 0.0f))));
    self->logic = ((self->safe != (self->a > self->b)) || (!(self->ticks == 3)));
}

int main(void) {
    MixedNumeric st;
    MixedNumeric_init(&st);
    for (int scan = 0; scan < 1; ++scan) {
        MixedNumeric_step(&st);
        printf("seed=%g\n", st.seed);
        printf("n=%d\n", st.n);
        printf("negative=%d\n", st.negative);
        printf("unit=%g\n", st.unit);
        printf("a=%g\n", st.a);
        printf("b=%g\n", st.b);
        printf("c=%g\n", st.c);
        printf("nested=%g\n", st.nested);
        printf("powf=%g\n", st.st_powf);
        printf("rounded=%d\n", st.rounded);
        printf("ticks=%d\n", st.ticks);
        printf("i=%d\n", st.i);
        printf("st2c_lhs=%d\n", st.st2c_lhs);
        printf("safe=%d\n", st.safe);
        printf("logic=%d\n", st.logic);
    }
    return 0;
}
