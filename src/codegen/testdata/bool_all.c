#include <stdint.h>
#include <stdio.h>

_Bool InRange(int16_t v, int16_t lo, int16_t hi);

int16_t Gate(_Bool en, int16_t v);

typedef struct {
    _Bool set;
    _Bool rst;
    _Bool q;
} Latch;
void Latch_init(Latch *self);
void Latch_body(Latch *self);

typedef struct {
    _Bool t;
    _Bool q;
} Toggle;
void Toggle_init(Toggle *self);
void Toggle_body(Toggle *self);

typedef struct {
    _Bool on;
    _Bool off;
    int16_t n;
    _Bool gt;
    _Bool both;
    _Bool either;
    _Bool differ;
    _Bool neither;
    _Bool prec;
    _Bool grouped;
    _Bool chain;
    int16_t cmp;
    _Bool inside;
    int16_t gated;
    int16_t i;
    _Bool seen;
    _Bool allSmall;
    int16_t cycle;
    _Bool armed;
    _Bool held;
    _Bool flipped;
    Latch sr;
    Toggle flip;
} BoolAll;
void BoolAll_init(BoolAll *self);
void BoolAll_step(BoolAll *self);

_Bool InRange(int16_t v, int16_t lo, int16_t hi) {
    _Bool InRange = 0;
    InRange = ((v >= lo) && (v <= hi));
    return InRange;
}

int16_t Gate(_Bool en, int16_t v) {
    int16_t Gate = 0;
    Gate = 0;
    if (en) {
        Gate = v;
    }
    return Gate;
}

void Latch_init(Latch *self) {
    self->set = 0;
    self->rst = 0;
    self->q = 0;
}

void Latch_body(Latch *self) {
    self->q = ((self->q || self->set) && (!self->rst));
}

void Toggle_init(Toggle *self) {
    self->t = 1;
    self->q = 0;
}

void Toggle_body(Toggle *self) {
    self->q = (self->q != self->t);
}

void BoolAll_init(BoolAll *self) {
    self->on = 1;
    self->off = 0;
    self->n = 7;
    self->gt = 0;
    self->both = 0;
    self->either = 0;
    self->differ = 0;
    self->neither = 0;
    self->prec = 0;
    self->grouped = 0;
    self->chain = 0;
    self->cmp = 0;
    self->inside = 0;
    self->gated = 0;
    self->i = 0;
    self->seen = 0;
    self->allSmall = 0;
    self->cycle = 0;
    self->armed = 0;
    self->held = 0;
    self->flipped = 0;
    Latch_init(&self->sr);
    Toggle_init(&self->flip);
}

void BoolAll_step(BoolAll *self) {
    self->gt = (self->n > 5);
    self->both = (self->on && self->gt);
    self->either = (self->off || self->gt);
    self->differ = (self->on != self->gt);
    self->neither = ((!self->off) && (!self->differ));
    self->prec = (self->on || (self->off && self->off));
    self->grouped = ((self->on || self->off) && self->off);
    self->chain = (self->on || (self->off != (self->gt && self->both)));
    self->cmp = 0;
    if (self->on) {
        self->cmp = (self->cmp + 1);
    }
    if ((!self->off)) {
        self->cmp = (self->cmp + 2);
    }
    if (((self->n > 5) && (self->n < 10))) {
        self->cmp = (self->cmp + 4);
    }
    if ((self->differ || (self->n == 0))) {
        self->cmp = (self->cmp + 8);
    }
    if ((self->gt == self->on)) {
        self->cmp = (self->cmp + 16);
    }
    if ((self->gt != self->off)) {
        self->cmp = (self->cmp + 32);
    }
    self->inside = InRange(self->n, 1, 10);
    self->gated = (Gate(self->inside, self->n) + Gate((!self->inside), 100));
    self->seen = 0;
    self->allSmall = 1;
    {
        int32_t __i0 = 1, __end0 = 10, __step0 = 1;
        for (; __i0 <= __end0; __i0 += __step0) {
            self->i = (int16_t)__i0;
            self->seen = (self->seen || (self->i == self->n));
            self->allSmall = (self->allSmall && (self->i < 5));
        }
    }
    self->cycle = (self->cycle + 1);
    self->armed = (!self->armed);
    self->sr.set = (self->cycle == 1);
    self->sr.rst = (self->cycle == 3);
    Latch_body(&self->sr);
    self->held = self->sr.q;
    Toggle_body(&self->flip);
    self->flipped = self->flip.q;
}

int main(void) {
    BoolAll st;
    BoolAll_init(&st);
    for (int scan = 0; scan < 3; ++scan) {
        BoolAll_step(&st);
        printf("on=%d\n", st.on);
        printf("off=%d\n", st.off);
        printf("n=%d\n", st.n);
        printf("gt=%d\n", st.gt);
        printf("both=%d\n", st.both);
        printf("either=%d\n", st.either);
        printf("differ=%d\n", st.differ);
        printf("neither=%d\n", st.neither);
        printf("prec=%d\n", st.prec);
        printf("grouped=%d\n", st.grouped);
        printf("chain=%d\n", st.chain);
        printf("cmp=%d\n", st.cmp);
        printf("inside=%d\n", st.inside);
        printf("gated=%d\n", st.gated);
        printf("i=%d\n", st.i);
        printf("seen=%d\n", st.seen);
        printf("allSmall=%d\n", st.allSmall);
        printf("cycle=%d\n", st.cycle);
        printf("armed=%d\n", st.armed);
        printf("held=%d\n", st.held);
        printf("flipped=%d\n", st.flipped);
    }
    return 0;
}
