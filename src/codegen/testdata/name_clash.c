#include <stdint.h>
#include <stdio.h>

typedef struct {
    int16_t st_switch;
    int16_t st_default;
    int16_t st_static;
    int16_t st_struct;
    int16_t st_self;
    int16_t st___end;
    int16_t st___step;
    int16_t i;
    int16_t total;
} NameClash;
void NameClash_init(NameClash *self);
void NameClash_step(NameClash *self);

void NameClash_init(NameClash *self) {
    self->st_switch = 0;
    self->st_default = 0;
    self->st_static = 0;
    self->st_struct = 1;
    self->st_self = 2;
    self->st___end = 3;
    self->st___step = 4;
    self->i = 0;
    self->total = 0;
}

void NameClash_step(NameClash *self) {
    self->st_switch = 10;
    self->st_default = 20;
    self->st_static = 30;
    self->total = ((((((self->st_switch + self->st_default) + self->st_static) + self->st_struct) + self->st_self) + self->st___end) + self->st___step);
    {
        int32_t __i0 = 1, __end0 = 3, __step0 = 1;
        for (; __i0 <= __end0; __i0 += __step0) {
            self->i = (int16_t)__i0;
            self->total = (self->total + self->st___step);
        }
    }
}

int main(void) {
    NameClash st;
    NameClash_init(&st);
    for (int scan = 0; scan < 1; ++scan) {
        NameClash_step(&st);
        printf("switch=%d\n", st.st_switch);
        printf("default=%d\n", st.st_default);
        printf("static=%d\n", st.st_static);
        printf("struct=%d\n", st.st_struct);
        printf("self=%d\n", st.st_self);
        printf("__end=%d\n", st.st___end);
        printf("__step=%d\n", st.st___step);
        printf("i=%d\n", st.i);
        printf("total=%d\n", st.total);
    }
    return 0;
}
