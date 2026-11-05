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

int16_t StopAt(int16_t n);

int16_t FirstSquare(float target);

typedef struct {
    int16_t stop;
    int16_t count;
    int16_t i;
    int16_t j;
} Sweep;
void Sweep_init(Sweep *self);
void Sweep_body(Sweep *self);

typedef struct {
    int16_t scan;
    int16_t first;
    int16_t again;
    int16_t total;
    int16_t found;
    int16_t tail;
    float scaled;
    _Bool valid;
    Sweep worker;
} MixedControl;
void MixedControl_init(MixedControl *self);
void MixedControl_step(MixedControl *self);

int16_t StopAt(int16_t n) {
    int16_t st2c_lhs, st2c_rhs;
    int16_t StopAt = 0;
    if (((st2c_lhs = n, st2c_rhs = 2, st2c_mod(st2c_lhs, st2c_rhs)) == 0)) {
        StopAt = 3;
    } else if ((n > 0)) {
        StopAt = 2;
    } else {
        StopAt = 1;
    }
    return StopAt;
}

int16_t FirstSquare(float target) {
    int16_t st2c_lhs1, st2c_rhs1;
    float st2c_lhs2, st2c_rhs2;
    int16_t FirstSquare = 0;
    int16_t k = 0;
    {
        int32_t __i0 = 1, __end0 = 5, __step0 = 1;
        for (; __i0 <= __end0; __i0 += __step0) {
            k = (int16_t)__i0;
            {
                int32_t st2c_sel = (st2c_lhs1 = k, st2c_rhs1 = 3, st2c_mod(st2c_lhs1, st2c_rhs1));
                switch (st2c_sel) {
                case 0:
                {
                    break;
                }
                default:
                {
                    if (((st2c_lhs2 = (float)k, st2c_rhs2 = 2, st2c_pow(st2c_lhs2, st2c_rhs2)) >= target)) {
                        FirstSquare = k;
                        return FirstSquare;
                    }
                    break;
                }
                }
            }
        }
    }
    return FirstSquare;
}

void Sweep_init(Sweep *self) {
    self->stop = 0;
    self->count = 0;
    self->i = 0;
    self->j = 0;
}

void Sweep_body(Sweep *self) {
    int16_t st2c_lhs3, st2c_rhs3;
    self->count = 0;
    {
        int32_t __i0 = 1, __end0 = 3, __step0 = 1;
        for (; __i0 <= __end0; __i0 += __step0) {
            self->i = (int16_t)__i0;
            self->j = 0;
            {
                int st2c_exit = 0;
                do {
                    self->j = (self->j + 1);
                    {
                        int32_t st2c_sel1 = (st2c_lhs3 = self->j, st2c_rhs3 = 3, st2c_mod(st2c_lhs3, st2c_rhs3));
                        switch (st2c_sel1) {
                        case 1:
                        {
                            self->count = (self->count + 1);
                            break;
                        }
                        case 2:
                        {
                            {
                                int st2c_exit1 = 0;
                                while (1) {
                                    {
                                        int32_t st2c_sel2 = self->i;
                                        int st2c_arm = 0;
                                        if ((st2c_sel2 >= 1 && st2c_sel2 <= 2)) st2c_arm = 1;
                                        switch (st2c_arm) {
                                        case 1:
                                        {
                                            self->count = (self->count + 10);
                                            st2c_exit1 = 1;
                                            break;
                                            break;
                                        }
                                        default:
                                        {
                                            self->count = (self->count + 100);
                                            st2c_exit1 = 1;
                                            break;
                                            break;
                                        }
                                        }
                                        if (st2c_exit1) break;
                                    }
                                }
                            }
                            {
                                int32_t st2c_sel3 = self->i;
                                switch (st2c_sel3) {
                                case 2:
                                {
                                    st2c_exit = 1;
                                    break;
                                    break;
                                }
                                }
                                if (st2c_exit) break;
                            }
                            break;
                        }
                        default:
                        {
                            st2c_exit = 1;
                            break;
                            break;
                        }
                        }
                        if (st2c_exit) break;
                    }
                    self->count = (self->count + 1);
                } while (!((self->j >= self->stop)));
            }
            if ((self->i >= self->stop)) {
                break;
            }
        }
    }
}

void MixedControl_init(MixedControl *self) {
    self->scan = 0;
    self->first = 0;
    self->again = 0;
    self->total = 0;
    self->found = 0;
    self->tail = 0;
    self->scaled = 0.0f;
    self->valid = 0;
    Sweep_init(&self->worker);
}

void MixedControl_step(MixedControl *self) {
    float st2c_lhs4, st2c_rhs4;
    int16_t st2c_lhs5, st2c_rhs5;
    self->scan = (self->scan + 1);
    self->worker.stop = StopAt(self->scan);
    Sweep_body(&self->worker);
    self->first = self->worker.count;
    self->worker.stop = 2;
    Sweep_body(&self->worker);
    self->again = self->worker.count;
    self->total = ((self->total + self->first) + self->again);
    self->found = FirstSquare((st2c_lhs4 = (float)self->scan, st2c_rhs4 = 2, st2c_pow(st2c_lhs4, st2c_rhs4)));
    self->scaled = (((float)(self->first)) / 4.0f);
    self->valid = (((self->again == 25) && (self->found > 0)) && (!((st2c_lhs5 = self->found, st2c_rhs5 = 3, st2c_mod(st2c_lhs5, st2c_rhs5)) == 0)));
    {
        int32_t st2c_sel4 = self->scan;
        switch (st2c_sel4) {
        case 2:
        {
            return;
            break;
        }
        }
    }
    self->tail = (self->tail + 1);
}

int main(void) {
    MixedControl st;
    MixedControl_init(&st);
    for (int scan = 0; scan < 3; ++scan) {
        MixedControl_step(&st);
        printf("scan=%d\n", st.scan);
        printf("first=%d\n", st.first);
        printf("again=%d\n", st.again);
        printf("total=%d\n", st.total);
        printf("found=%d\n", st.found);
        printf("tail=%d\n", st.tail);
        printf("scaled=%g\n", st.scaled);
        printf("valid=%d\n", st.valid);
    }
    return 0;
}
