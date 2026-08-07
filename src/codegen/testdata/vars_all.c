#include <stdint.h>
#include <stdio.h>

typedef struct {
    int16_t LIMIT;
    int16_t STEP_SIZE;
    int16_t saved;
    int16_t a;
    int16_t b;
    int16_t c;
    int16_t result;
    int16_t scratch;
    int16_t untouched;
} VarsAll;
void VarsAll_init(VarsAll *self);
void VarsAll_step(VarsAll *self);

void VarsAll_init(VarsAll *self) {
    self->LIMIT = 100;
    self->STEP_SIZE = 3;
    self->saved = (-5);
    self->a = 7;
    self->b = 7;
    self->c = 7;
    self->result = 0;
    self->scratch = 0;
    self->untouched = 0;
}

void VarsAll_step(VarsAll *self) {
    self->scratch = ((self->a + self->b) + self->c);
    self->saved = (self->saved + self->STEP_SIZE);
    self->result = (((self->scratch + self->saved) + self->LIMIT) + self->untouched);
}

int main(void) {
    VarsAll st;
    VarsAll_init(&st);
    for (int scan = 0; scan < 1; ++scan) {
        VarsAll_step(&st);
        printf("LIMIT=%d\n", st.LIMIT);
        printf("STEP_SIZE=%d\n", st.STEP_SIZE);
        printf("saved=%d\n", st.saved);
        printf("a=%d\n", st.a);
        printf("b=%d\n", st.b);
        printf("c=%d\n", st.c);
        printf("result=%d\n", st.result);
        printf("scratch=%d\n", st.scratch);
        printf("untouched=%d\n", st.untouched);
    }
    return 0;
}
