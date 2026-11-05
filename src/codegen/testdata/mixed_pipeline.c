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

typedef struct {
    float delta;
    float total;
} Bucket;
void Bucket_init(Bucket *self);
void Bucket_body(Bucket *self);

typedef struct {
    float step;
    _Bool enable;
    float total;
    _Bool ready;
    Bucket inner;
} Channel;
void Channel_init(Channel *self);
void Channel_body(Channel *self);

typedef struct {
    int16_t scan;
    float increment;
    float leftTotal;
    float rightTotal;
    float combined;
    _Bool enabled;
    _Bool ready;
    Channel left;
    Channel right;
    Bucket sink;
} MixedPipeline;
void MixedPipeline_init(MixedPipeline *self);
void MixedPipeline_step(MixedPipeline *self);

float Sample(int16_t n, float gain);

void MixedPipeline_init(MixedPipeline *self) {
    self->scan = 0;
    self->increment = 0.0f;
    self->leftTotal = 0.0f;
    self->rightTotal = 0.0f;
    self->combined = 0.0f;
    self->enabled = 0;
    self->ready = 0;
    Channel_init(&self->left);
    Channel_init(&self->right);
    Bucket_init(&self->sink);
}

void MixedPipeline_step(MixedPipeline *self) {
    int16_t st2c_lhs, st2c_rhs;
    self->scan = (self->scan + 1);
    self->enabled = ((st2c_lhs = (+self->scan), st2c_rhs = 3, st2c_mod(st2c_lhs, st2c_rhs)) != 0);
    self->increment = Sample(self->scan, (1.0f / 2.0f));
    if ((self->scan == 1)) {
        self->right.step = (1.0f / 2.0f);
    }
    self->sink.delta = (-100.0f);
    self->left.step = self->increment;
    self->left.enable = self->enabled;
    Channel_body(&self->left);
    self->sink.delta = self->left.total;
    self->leftTotal = self->left.total;
    Bucket_body(&self->sink);
    self->right.enable = (!self->enabled);
    Channel_body(&self->right);
    self->rightTotal = self->right.total;
    self->combined = (self->sink.total + self->rightTotal);
    self->ready = (self->left.ready != self->right.ready);
}

void Channel_init(Channel *self) {
    self->step = 0.0f;
    self->enable = 1;
    self->total = 0.0f;
    self->ready = 0;
    Bucket_init(&self->inner);
}

void Channel_body(Channel *self) {
    if ((!self->enable)) {
        self->ready = 0;
        return;
    }
    self->inner.delta = self->step;
    Bucket_body(&self->inner);
    self->total = self->inner.total;
    self->ready = (self->total >= 1.0f);
}

float Sample(int16_t n, float gain) {
    int16_t st2c_lhs1, st2c_rhs1;
    float st2c_lhs2, st2c_rhs2;
    float Sample = 0.0f;
    {
        int32_t st2c_sel = (st2c_lhs1 = n, st2c_rhs1 = 3, st2c_mod(st2c_lhs1, st2c_rhs1));
        switch (st2c_sel) {
        case 0:
        {
            return Sample;
            break;
        }
        case 1:
        {
            Sample = (st2c_lhs2 = (((float)(n)) * gain), st2c_rhs2 = 2, st2c_pow(st2c_lhs2, st2c_rhs2));
            return Sample;
            break;
        }
        case 2:
        {
            Sample = (((float)(n)) * gain);
            break;
        }
        }
    }
    return Sample;
}

void Bucket_init(Bucket *self) {
    self->delta = 0.0f;
    self->total = 0.0f;
}

void Bucket_body(Bucket *self) {
    self->total = (self->total + self->delta);
}

int main(void) {
    MixedPipeline st;
    MixedPipeline_init(&st);
    for (int scan = 0; scan < 4; ++scan) {
        MixedPipeline_step(&st);
        printf("scan=%d\n", st.scan);
        printf("increment=%g\n", st.increment);
        printf("leftTotal=%g\n", st.leftTotal);
        printf("rightTotal=%g\n", st.rightTotal);
        printf("combined=%g\n", st.combined);
        printf("enabled=%d\n", st.enabled);
        printf("ready=%d\n", st.ready);
    }
    return 0;
}
