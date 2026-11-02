#include <stdint.h>
#include <stdio.h>

int16_t Pick(int16_t n);

typedef struct {
    _Bool shortStep;
    int16_t count;
} Meter;
void Meter_init(Meter *self);
void Meter_body(Meter *self);

typedef struct {
    Meter meterInst;
    int16_t scan;
    int16_t picked;
    int16_t seen;
    int16_t tail;
} OpsReturn;
void OpsReturn_init(OpsReturn *self);
void OpsReturn_step(OpsReturn *self);

int16_t Pick(int16_t n) {
    int16_t Pick = 0;
    if ((n == 1)) {
        return Pick;
    }
    Pick = (n * 2);
    return Pick;
    Pick = 99;
    return Pick;
}

void Meter_init(Meter *self) {
    self->shortStep = 0;
    self->count = 0;
}

void Meter_body(Meter *self) {
    self->count = (self->count + 1);
    if (self->shortStep) {
        return;
    }
    self->count = (self->count + 10);
}

void OpsReturn_init(OpsReturn *self) {
    Meter_init(&self->meterInst);
    self->scan = 0;
    self->picked = 0;
    self->seen = 0;
    self->tail = 0;
}

void OpsReturn_step(OpsReturn *self) {
    self->scan = (self->scan + 1);
    self->picked = Pick(self->scan);
    self->meterInst.shortStep = (self->scan != 2);
    Meter_body(&self->meterInst);
    self->seen = self->meterInst.count;
    if ((self->scan == 2)) {
        return;
    }
    self->tail = (self->tail + 1);
}

int main(void) {
    OpsReturn st;
    OpsReturn_init(&st);
    for (int scan = 0; scan < 3; ++scan) {
        OpsReturn_step(&st);
        printf("scan=%d\n", st.scan);
        printf("picked=%d\n", st.picked);
        printf("seen=%d\n", st.seen);
        printf("tail=%d\n", st.tail);
    }
    return 0;
}
