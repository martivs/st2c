package codegen

import "strings"

const powerDefinition = `
static float %s(float base, float exponent) {
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
`

// C99 names exposed by the power helper's headers, including cases where
// stdio.h is included without a generated main.
var powerHeaderNames = func() map[string]bool {
	names := map[string]bool{}
	for _, name := range strings.Fields(`
		float_t double_t HUGE_VAL HUGE_VALF HUGE_VALL INFINITY NAN
		FP_INFINITE FP_NAN FP_NORMAL FP_SUBNORMAL FP_ZERO FP_ILOGB0 FP_ILOGBNAN
		MATH_ERRNO MATH_ERREXCEPT math_errhandling fpclassify isfinite isinf
		isnan isnormal signbit isgreater isgreaterequal isless islessequal
		islessgreater isunordered FP_FAST_FMA FP_FAST_FMAF FP_FAST_FMAL
		size_t wchar_t NULL div_t ldiv_t lldiv_t EXIT_FAILURE EXIT_SUCCESS
		RAND_MAX MB_CUR_MAX atof atoi atol atoll strtod strtof strtold strtol
		strtoll strtoul strtoull rand srand calloc free malloc realloc abort
		atexit exit _Exit getenv system bsearch qsort abs labs llabs div ldiv
		lldiv mblen mbtowc wctomb mbstowcs wcstombs
		FILE fpos_t EOF BUFSIZ FOPEN_MAX FILENAME_MAX L_tmpnam TMP_MAX
		_IOFBF _IOLBF _IONBF SEEK_CUR SEEK_END SEEK_SET stderr stdin stdout
		remove rename tmpfile tmpnam fclose fflush fopen freopen setbuf setvbuf
		fprintf fscanf printf scanf snprintf sprintf sscanf vfprintf vfscanf
		vprintf vscanf vsnprintf vsprintf vsscanf fgetc fgets fputc fputs getc
		getchar gets putc putchar puts ungetc fread fwrite fgetpos fseek fsetpos
		ftell rewind clearerr feof ferror perror
	`) {
		names[name] = true
	}
	for _, stem := range strings.Fields(`
		acos asin atan atan2 cos sin tan acosh asinh atanh cosh sinh tanh
		exp exp2 expm1 frexp ilogb ldexp log log10 log1p log2 logb modf scalbn
		scalbln cbrt fabs hypot pow sqrt erf erfc lgamma tgamma ceil floor
		nearbyint rint lrint llrint round lround llround trunc fmod remainder
		remquo copysign nan nextafter nexttoward fdim fmax fmin fma
	`) {
		for _, suffix := range []string{"", "f", "l"} {
			names[stem+suffix] = true
		}
	}
	return names
}()

// Rename only in translation units using power, after collecting all mapped
// user names. Preserve the original ST spelling for lookup and driver output.
func (g *gen) protectPowerNames(units []unit) {
	rename := func(name string) string {
		if powerHeaderNames[name] {
			return g.freshName("st_" + name)
		}
		return name
	}
	for _, u := range units {
		if s := u.state; s != nil {
			if powerHeaderNames[s.cName] {
				base := "st_" + s.cName
				for {
					name := g.freshName(base)
					if !g.names[name+"_init"] && !g.names[name+"_"+s.stepName] {
						s.cName = name
						g.names[name+"_init"] = true
						g.names[name+"_"+s.stepName] = true
						break
					}
				}
			}
			for _, v := range s.order {
				v.cName = rename(v.cName)
			}
			continue
		}
		fn := u.fn
		fn.cName = rename(fn.cName)
		fn.retVar.cName = fn.cName
		for _, block := range fn.f.VarBlocks {
			for _, decl := range block.Decls {
				for _, id := range decl.Names {
					v := fn.vars[strings.ToUpper(id.Name)]
					v.cName = rename(v.cName)
				}
			}
		}
	}
}
