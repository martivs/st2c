#include <stdint.h>
#include <stdio.h>

static int16_t st_real_to_int(float v) {
    return (int16_t)(v >= 0.0f ? v + 0.5f : v - 0.5f);
}

float Half(float x);

float Scale(float v, int16_t k);

typedef struct {
    float v;
    float total;
    float dt;
} Integrator;
void Integrator_init(Integrator *self);
void Integrator_body(Integrator *self);

typedef struct {
    float a;
    float b;
    float neg;
    float big;
    float zero;
    float quot;
    float half;
    float scaled;
    float conv;
    int16_t rounded;
    int16_t negRounded;
    int16_t cmp;
    int16_t i;
    float acc;
    float total;
    Integrator integ;
} RealAll;
void RealAll_init(RealAll *self);
void RealAll_step(RealAll *self);

float Half(float x) {
    float Half = 0.0f;
    Half = (x / 2.0f);
    return Half;
}

float Scale(float v, int16_t k) {
    float Scale = 0.0f;
    Scale = (v * ((float)(k)));
    return Scale;
}

void Integrator_init(Integrator *self) {
    self->v = 0.0f;
    self->total = 0.0f;
    self->dt = 0.25f;
}

void Integrator_body(Integrator *self) {
    self->total = (self->total + (self->v * self->dt));
}

void RealAll_init(RealAll *self) {
    self->a = 2.5f;
    self->b = 0.25f;
    self->neg = (-1.5f);
    self->big = 1500.0f;
    self->zero = 0.0f;
    self->quot = 0.0f;
    self->half = 0.0f;
    self->scaled = 0.0f;
    self->conv = 0.0f;
    self->rounded = 0;
    self->negRounded = 0;
    self->cmp = 0;
    self->i = 0;
    self->acc = 0.0f;
    self->total = 0.0f;
    Integrator_init(&self->integ);
}

void RealAll_step(RealAll *self) {
    self->quot = (1.0f / 2.0f);
    self->half = Half((self->a + 1.0f));
    self->scaled = Scale(self->b, 3);
    self->rounded = st_real_to_int(self->a);
    self->negRounded = st_real_to_int(self->neg);
    self->conv = (((float)(self->rounded)) - self->b);
    self->cmp = 0;
    if ((self->a > self->b)) {
        self->cmp = (self->cmp + 1);
    }
    if (((self->a * self->b) == 0.625f)) {
        self->cmp = (self->cmp + 2);
    }
    if ((self->zero < 1.0f)) {
        self->cmp = (self->cmp + 4);
    }
    if ((self->neg >= 0.0f)) {
        self->cmp = (self->cmp + 8);
    }
    if ((self->b != 0.25f)) {
        self->cmp = (self->cmp + 16);
    }
    self->acc = 0.0f;
    {
        int32_t __i0 = 1, __end0 = 4, __step0 = 1;
        for (; __i0 <= __end0; __i0 += __step0) {
            self->i = (int16_t)__i0;
            self->acc = (self->acc + (((float)(self->i)) * 0.5f));
        }
    }
    self->integ.v = self->a;
    Integrator_body(&self->integ);
    self->integ.v = self->b;
    Integrator_body(&self->integ);
    self->total = self->integ.total;
}

int main(void) {
    RealAll st;
    RealAll_init(&st);
    for (int scan = 0; scan < 1; ++scan) {
        RealAll_step(&st);
        printf("a=%g\n", st.a);
        printf("b=%g\n", st.b);
        printf("neg=%g\n", st.neg);
        printf("big=%g\n", st.big);
        printf("zero=%g\n", st.zero);
        printf("quot=%g\n", st.quot);
        printf("half=%g\n", st.half);
        printf("scaled=%g\n", st.scaled);
        printf("conv=%g\n", st.conv);
        printf("rounded=%d\n", st.rounded);
        printf("negRounded=%d\n", st.negRounded);
        printf("cmp=%d\n", st.cmp);
        printf("i=%d\n", st.i);
        printf("acc=%g\n", st.acc);
        printf("total=%g\n", st.total);
    }
    return 0;
}
