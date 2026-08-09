#include <stdint.h>
#include <stdio.h>

typedef struct {
    int16_t step;
    int16_t count;
    int16_t calls;
} Counter;
void Counter_init(Counter *self);
void Counter_body(Counter *self);

typedef struct {
    Counter fast;
    Counter slow;
    int16_t fastOut;
    int16_t total;
} FbCounter;
void FbCounter_init(FbCounter *self);
void FbCounter_step(FbCounter *self);

void Counter_init(Counter *self) {
    self->step = 0;
    self->count = 0;
    self->calls = 0;
}

void Counter_body(Counter *self) {
    self->calls = (self->calls + 1);
    self->count = (self->count + self->step);
}

void FbCounter_init(FbCounter *self) {
    Counter_init(&self->fast);
    Counter_init(&self->slow);
    self->fastOut = 0;
    self->total = 0;
}

void FbCounter_step(FbCounter *self) {
    self->fast.step = 10;
    Counter_body(&self->fast);
    self->fastOut = self->fast.count;
    self->slow.step = 1;
    Counter_body(&self->slow);
    self->total = (self->fastOut + self->slow.count);
}

int main(void) {
    FbCounter st;
    FbCounter_init(&st);
    for (int scan = 0; scan < 3; ++scan) {
        FbCounter_step(&st);
        printf("fastOut=%d\n", st.fastOut);
        printf("total=%d\n", st.total);
    }
    return 0;
}
