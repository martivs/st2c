#include <stdint.h>
#include <stdio.h>

typedef struct {
    int16_t add;
    int16_t sum;
} Accum;
void Accum_init(Accum *self);
void Accum_body(Accum *self);

typedef struct {
    int16_t tick;
    int16_t doubled;
    Accum inner;
    int16_t seed;
} Machine;
void Machine_init(Machine *self);
void Machine_body(Machine *self);

typedef struct {
    Machine m;
    int16_t result;
} FbNested;
void FbNested_init(FbNested *self);
void FbNested_step(FbNested *self);

void Accum_init(Accum *self) {
    self->add = 0;
    self->sum = 0;
}

void Accum_body(Accum *self) {
    self->sum = (self->sum + self->add);
}

void Machine_init(Machine *self) {
    self->tick = 0;
    self->doubled = 0;
    Accum_init(&self->inner);
    self->seed = 100;
}

void Machine_body(Machine *self) {
    self->inner.add = self->tick;
    Accum_body(&self->inner);
    self->doubled = ((self->inner.sum * 2) + self->seed);
}

void FbNested_init(FbNested *self) {
    Machine_init(&self->m);
    self->result = 0;
}

void FbNested_step(FbNested *self) {
    self->m.tick = 5;
    Machine_body(&self->m);
    self->result = self->m.doubled;
}

int main(void) {
    FbNested st;
    FbNested_init(&st);
    for (int scan = 0; scan < 3; ++scan) {
        FbNested_step(&st);
        printf("result=%d\n", st.result);
    }
    return 0;
}
