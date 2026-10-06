#include <stdint.h>
#include <stdio.h>

int16_t Digits(int16_t v);

typedef struct {
    int16_t inflow;
    int16_t level;
    int16_t steps;
} Drain;
void Drain_init(Drain *self);
void Drain_body(Drain *self);

int16_t StepsTo(int16_t limit);

_Bool Ripe(int16_t v);

typedef struct {
    int16_t target;
    int16_t speed;
    int16_t ticks;
    _Bool atTop;
} Ramp;
void Ramp_init(Ramp *self);
void Ramp_body(Ramp *self);

typedef struct {
    int16_t cycle;
    int16_t t;
    int16_t first;
    int16_t mode;
    int16_t band;
    int16_t sign;
    int16_t legacy;
    int16_t whileZero;
    int16_t repeatOne;
    int16_t sum;
    int16_t k;
    int16_t rep;
    int16_t found;
    int16_t i;
    int16_t j;
    int16_t outerCnt;
    int16_t innerCnt;
    int16_t spins;
    int16_t acc;
    float r;
    int16_t halves;
    _Bool flag;
    int16_t tries;
    int16_t digitsZero;
    int16_t digitsBig;
    Drain tank;
    int16_t lvl;
    int16_t drained;
    int16_t pick;
    int16_t taps;
    int16_t wOuter;
    int16_t wInner;
    int16_t wStep;
    int16_t rOuter;
    int16_t rInner;
    int16_t rStep;
    int16_t stepsUsed;
    int16_t grown;
    _Bool armed;
    int16_t paces;
    Ramp ramp;
    int16_t rampSpeed;
    int16_t rampTicks;
    _Bool rampTop;
    int16_t gear;
} CtrlAll;
void CtrlAll_init(CtrlAll *self);
void CtrlAll_step(CtrlAll *self);

int16_t Digits(int16_t v) {
    int16_t Digits = 0;
    int16_t n = 0;
    int16_t c = 0;
    n = v;
    c = 0;
    do {
        c = (c + 1);
        n = (n / 10);
    } while (!((n == 0)));
    Digits = c;
    return Digits;
}

void Drain_init(Drain *self) {
    self->inflow = 0;
    self->level = 0;
    self->steps = 0;
}

void Drain_body(Drain *self) {
    self->level = (self->level + self->inflow);
    self->steps = 0;
    while ((self->level > 10)) {
        self->level = (self->level - 4);
        self->steps = (self->steps + 1);
    }
}

int16_t StepsTo(int16_t limit) {
    int16_t StepsTo = 0;
    int16_t n = 0;
    int16_t acc = 0;
    while ((n < 100)) {
        n = (n + 1);
        acc = (acc + n);
        if ((acc > limit)) {
            StepsTo = n;
            break;
        }
    }
    return StepsTo;
}

_Bool Ripe(int16_t v) {
    _Bool Ripe = 0;
    Ripe = (v >= 4);
    return Ripe;
}

void Ramp_init(Ramp *self) {
    self->target = 0;
    self->speed = 0;
    self->ticks = 0;
    self->atTop = 0;
}

void Ramp_body(Ramp *self) {
    self->ticks = 0;
    do {
        if ((self->ticks >= 2)) {
            break;
        }
        self->speed = (self->speed + 3);
        self->ticks = (self->ticks + 1);
    } while (!((self->speed >= self->target)));
    self->atTop = (self->speed >= self->target);
}

void CtrlAll_init(CtrlAll *self) {
    self->cycle = 0;
    self->t = 5;
    self->first = 0;
    self->mode = 0;
    self->band = 99;
    self->sign = 0;
    self->legacy = 0;
    self->whileZero = 0;
    self->repeatOne = 0;
    self->sum = 0;
    self->k = 0;
    self->rep = 0;
    self->found = 0;
    self->i = 0;
    self->j = 0;
    self->outerCnt = 0;
    self->innerCnt = 0;
    self->spins = 0;
    self->acc = 0;
    self->r = 0.0f;
    self->halves = 0;
    self->flag = 0;
    self->tries = 0;
    self->digitsZero = 0;
    self->digitsBig = 0;
    Drain_init(&self->tank);
    self->lvl = 0;
    self->drained = 0;
    self->pick = 0;
    self->taps = 0;
    self->wOuter = 0;
    self->wInner = 0;
    self->wStep = 0;
    self->rOuter = 0;
    self->rInner = 0;
    self->rStep = 0;
    self->stepsUsed = 0;
    self->grown = 0;
    self->armed = 0;
    self->paces = 0;
    Ramp_init(&self->ramp);
    self->rampSpeed = 0;
    self->rampTicks = 0;
    self->rampTop = 0;
    self->gear = 0;
}

void CtrlAll_step(CtrlAll *self) {
    self->cycle = (self->cycle + 1);
    if ((self->t < 10)) {
        self->first = 1;
    } else if ((self->t < 20)) {
        self->first = 2;
    } else if ((self->t < 30)) {
        self->first = 3;
    }
    if ((self->cycle == 1)) {
        self->mode = 10;
    } else if ((self->cycle == 2)) {
        self->mode = 20;
    } else {
        self->mode = 30;
    }
    if ((self->t > 100)) {
        self->band = 1;
    } else if ((self->t > 50)) {
        self->band = 2;
    }
    if ((self->t > 10)) {
        self->sign = 1;
    } else if ((self->t > 0)) {
        if ((self->t > 3)) {
            self->sign = 2;
        } else {
            self->sign = 3;
        }
    } else {
        self->sign = 4;
    }
    if ((self->t > 10)) {
        self->legacy = 1;
    } else {
        if ((self->t > 3)) {
            self->legacy = 2;
        }
    }
    self->whileZero = 0;
    self->k = 0;
    while ((self->k > 0)) {
        self->whileZero = (self->whileZero + 1);
        self->k = (self->k - 1);
    }
    self->repeatOne = 0;
    do {
        self->repeatOne = (self->repeatOne + 1);
    } while (!((self->repeatOne < 100)));
    self->sum = 0;
    self->k = 0;
    while ((self->k < 5)) {
        self->k = (self->k + 1);
        self->sum = (self->sum + self->k);
    }
    self->rep = 0;
    do {
        self->rep = (self->rep + 1);
        if ((self->rep == 3)) {
            break;
        }
    } while (!((self->rep >= 10)));
    self->found = 0;
    {
        int32_t __i0 = 1, __end0 = 10, __step0 = 1;
        for (; __i0 <= __end0; __i0 += __step0) {
            self->i = (int16_t)__i0;
            if (((self->i * self->i) > 20)) {
                self->found = self->i;
                break;
            }
        }
    }
    self->outerCnt = 0;
    self->innerCnt = 0;
    {
        int32_t __i1 = 1, __end1 = 3, __step1 = 1;
        for (; __i1 <= __end1; __i1 += __step1) {
            self->i = (int16_t)__i1;
            self->outerCnt = (self->outerCnt + 1);
            {
                int32_t __i2 = 1, __end2 = 10, __step2 = 1;
                for (; __i2 <= __end2; __i2 += __step2) {
                    self->j = (int16_t)__i2;
                    if ((self->j > 2)) {
                        break;
                    }
                    self->innerCnt = (self->innerCnt + 1);
                }
            }
        }
    }
    self->spins = 0;
    while (1) {
        self->spins = (self->spins + 1);
        if ((self->spins >= 4)) {
            break;
        }
    }
    self->acc = 0;
    self->k = 0;
    while ((self->k < 3)) {
        self->k = (self->k + 1);
        {
            int32_t __i3 = 1, __end3 = self->k, __step3 = 1;
            for (; __i3 <= __end3; __i3 += __step3) {
                self->j = (int16_t)__i3;
                self->acc = (self->acc + self->j);
            }
        }
    }
    self->r = 10.0f;
    self->halves = 0;
    while ((self->r > 1.0f)) {
        self->r = (self->r / 2.0f);
        self->halves = (self->halves + 1);
    }
    self->flag = 0;
    self->tries = 0;
    while (((!self->flag) && (self->tries < 10))) {
        self->tries = (self->tries + 1);
        self->flag = ((self->tries * 3) > 10);
    }
    self->digitsZero = Digits(0);
    self->digitsBig = Digits(12345);
    self->tank.inflow = 15;
    Drain_body(&self->tank);
    self->lvl = self->tank.level;
    self->drained = self->tank.steps;
    self->pick = 0;
    self->taps = 0;
    while ((self->pick < 9)) {
        self->pick = (self->pick + 1);
        if ((self->pick == 1)) {
            self->taps = (self->taps + 1);
        } else if ((self->pick == 2)) {
            self->taps = (self->taps + 10);
        } else if ((self->pick == 3)) {
            break;
        } else {
            self->taps = (self->taps + 100);
        }
    }
    self->wOuter = 0;
    self->wInner = 0;
    while ((self->wOuter < 4)) {
        self->wOuter = (self->wOuter + 1);
        self->wStep = 0;
        while ((self->wStep < 10)) {
            self->wStep = (self->wStep + 1);
            self->wInner = (self->wInner + 1);
            if ((self->wStep >= 3)) {
                break;
            }
        }
    }
    self->rOuter = 0;
    self->rInner = 0;
    do {
        self->rOuter = (self->rOuter + 1);
        self->rStep = 0;
        do {
            self->rStep = (self->rStep + 1);
            self->rInner = (self->rInner + 1);
            if ((self->rStep >= 3)) {
                break;
            }
        } while (!((self->rStep >= 9)));
    } while (!((self->rOuter >= 3)));
    self->stepsUsed = StepsTo(20);
    self->grown = 0;
    do {
        self->grown = (self->grown + 1);
    } while (!(Ripe(self->grown)));
    self->armed = 1;
    self->paces = 0;
    while (self->armed) {
        self->paces = (self->paces + 1);
        self->armed = (self->paces < 3);
    }
    self->ramp.target = 10;
    Ramp_body(&self->ramp);
    self->rampSpeed = self->ramp.speed;
    self->rampTicks = self->ramp.ticks;
    self->rampTop = self->ramp.atTop;
    if (self->armed) {
        self->gear = 9;
    } else if (self->ramp.atTop) {
        self->gear = 1;
    } else if (Ripe(self->rampSpeed)) {
        self->gear = 2;
    } else {
        self->gear = 3;
    }
}

int main(void) {
    CtrlAll st;
    CtrlAll_init(&st);
    for (int scan = 0; scan < 3; ++scan) {
        CtrlAll_step(&st);
        printf("cycle=%d\n", st.cycle);
        printf("t=%d\n", st.t);
        printf("first=%d\n", st.first);
        printf("mode=%d\n", st.mode);
        printf("band=%d\n", st.band);
        printf("sign=%d\n", st.sign);
        printf("legacy=%d\n", st.legacy);
        printf("whileZero=%d\n", st.whileZero);
        printf("repeatOne=%d\n", st.repeatOne);
        printf("sum=%d\n", st.sum);
        printf("k=%d\n", st.k);
        printf("rep=%d\n", st.rep);
        printf("found=%d\n", st.found);
        printf("i=%d\n", st.i);
        printf("j=%d\n", st.j);
        printf("outerCnt=%d\n", st.outerCnt);
        printf("innerCnt=%d\n", st.innerCnt);
        printf("spins=%d\n", st.spins);
        printf("acc=%d\n", st.acc);
        printf("r=%g\n", st.r);
        printf("halves=%d\n", st.halves);
        printf("flag=%d\n", st.flag);
        printf("tries=%d\n", st.tries);
        printf("digitsZero=%d\n", st.digitsZero);
        printf("digitsBig=%d\n", st.digitsBig);
        printf("lvl=%d\n", st.lvl);
        printf("drained=%d\n", st.drained);
        printf("pick=%d\n", st.pick);
        printf("taps=%d\n", st.taps);
        printf("wOuter=%d\n", st.wOuter);
        printf("wInner=%d\n", st.wInner);
        printf("wStep=%d\n", st.wStep);
        printf("rOuter=%d\n", st.rOuter);
        printf("rInner=%d\n", st.rInner);
        printf("rStep=%d\n", st.rStep);
        printf("stepsUsed=%d\n", st.stepsUsed);
        printf("grown=%d\n", st.grown);
        printf("armed=%d\n", st.armed);
        printf("paces=%d\n", st.paces);
        printf("rampSpeed=%d\n", st.rampSpeed);
        printf("rampTicks=%d\n", st.rampTicks);
        printf("rampTop=%d\n", st.rampTop);
        printf("gear=%d\n", st.gear);
    }
    return 0;
}
