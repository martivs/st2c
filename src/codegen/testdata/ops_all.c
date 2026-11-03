#include <stdint.h>
#include <math.h>
#include <stdlib.h>
#include <stdio.h>

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

float Choose(int16_t n);

typedef struct {
    float value;
    float total;
} Accumulate;
void Accumulate_init(Accumulate *self);
void Accumulate_body(Accumulate *self);

typedef struct {
    Accumulate counter;
    int16_t scan;
    float value;
    float total;
    int16_t i;
    int16_t marks;
} OpsAll;
void OpsAll_init(OpsAll *self);
void OpsAll_step(OpsAll *self);

float Choose(int16_t n) {
    int16_t st2c_lhs, st2c_rhs;
    float st2c_lhs1, st2c_rhs1;
    float Choose = 0.0f;
    {
        int32_t st2c_sel = (st2c_lhs = (+n), st2c_rhs = 3, st2c_mod(st2c_lhs, st2c_rhs));
        int st2c_arm = 0;
        if ((st2c_sel >= 1 && st2c_sel <= 2)) st2c_arm = 1;
        switch (st2c_arm) {
        case 1:
        {
            Choose = (st2c_lhs1 = ((float)(n)), st2c_rhs1 = 2, st2c_pow(st2c_lhs1, st2c_rhs1));
            return Choose;
            break;
        }
        default:
        {
            Choose = 0.5f;
            break;
        }
        }
    }
    return Choose;
}

void Accumulate_init(Accumulate *self) {
    self->value = 0.0f;
    self->total = 0.0f;
}

void Accumulate_body(Accumulate *self) {
    self->total = (self->total + self->value);
}

void OpsAll_init(OpsAll *self) {
    Accumulate_init(&self->counter);
    self->scan = 0;
    self->value = 0.0f;
    self->total = 0.0f;
    self->i = 0;
    self->marks = 0;
}

void OpsAll_step(OpsAll *self) {
    self->scan = (self->scan + 1);
    self->value = Choose(self->scan);
    self->counter.value = self->value;
    Accumulate_body(&self->counter);
    self->total = self->counter.total;
    self->marks = 0;
    {
        int st2c_exit = 0;
        {
            int32_t __i0 = 1, __end0 = 3, __step0 = 1;
            for (; __i0 <= __end0; __i0 += __step0) {
                self->i = (int16_t)__i0;
                {
                    int32_t st2c_sel1 = self->i;
                    switch (st2c_sel1) {
                    case 1:
                    {
                        do {
                            self->marks = (self->marks + 1);
                            break;
                        } while (!(1));
                        break;
                    }
                    case 2:
                    {
                        st2c_exit = 1;
                        break;
                        break;
                    }
                    }
                    if (st2c_exit) break;
                }
                self->marks = (self->marks + 10);
            }
        }
    }
}

int main(void) {
    OpsAll st;
    OpsAll_init(&st);
    for (int scan = 0; scan < 3; ++scan) {
        OpsAll_step(&st);
        printf("scan=%d\n", st.scan);
        printf("value=%g\n", st.value);
        printf("total=%g\n", st.total);
        printf("i=%d\n", st.i);
        printf("marks=%d\n", st.marks);
    }
    return 0;
}
