package jinja

import (
	"reflect"
	"strings"
)

func registerBuiltinTests(env *Environment) {
	tests := map[string]TestFunc{
		"defined":      testDefined,
		"undefined":    testUndefined,
		"none":         testNone,
		"string":       testString,
		"number":       testNumber,
		"integer":      testNumber,
		"float":        testFloat,
		"callable":     testCallable,
		"iterable":     testIterable,
		"mapping":      testMapping,
		"sequence":     testSequence,
		"sameas":       testSameAs,
		"divisibleby":  testDivisibleBy,
		"even":         testEven,
		"odd":          testOdd,
		"lower":        testLower,
		"upper":        testUpper,
		"startswith":   testStartsWith,
		"endswith":     testEndsWith,
		"in":           testIn,
		"equalto":      testEqualTo,
		"gt":           testGt,
		"gte":          testGte,
		"lt":           testLt,
		"lte":          testLte,
		"all":          testAll,
		"any":          testAny,
		"filter":       testFilter,
		"boolean":      testBoolean,
		"true":         testBoolean,
		"false":        testFalse,
		"namespace":    testNamespace,
		"escaped":      testEscaped,
		"containment":  testContainment,
		"matching":     testMatching,
		"subset":       testSubset,
		"superset":     testSuperset,
	}

	for name, fn := range tests {
		env.testFuncs[name] = fn
	}
}

func testDefined(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	return !IsUndefined(v) && v != nil, nil
}

func testUndefined(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	return IsUndefined(v) || v == nil, nil
}

func testNone(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	return v == nil, nil
}

func testString(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	_, ok := v.(string)
	return ok, nil
}

func testNumber(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	return isNumeric(v), nil
}

func testFloat(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	_, ok := v.(float64)
	return ok, nil
}

func testCallable(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	switch v.(type) {
	case func(...interface{}) interface{},
		func(...interface{}) (interface{}, error),
		func(*ExecutionContext, ...interface{}) (interface{}, error),
		*Macro,
		*GoFunc:
		return true, nil
	}
	rv := reflect.ValueOf(v)
	return v != nil && rv.Kind() == reflect.Func, nil
}

func testIterable(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if v == nil {
		return false, nil
	}
	switch v.(type) {
	case string, []interface{}, map[string]interface{}:
		return true, nil
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array || rv.Kind() == reflect.Map, nil
}

func testMapping(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	_, ok := v.(map[string]interface{})
	if ok {
		return true, nil
	}
	rv := reflect.ValueOf(v)
	return v != nil && rv.Kind() == reflect.Map, nil
}

func testSequence(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	_, ok := v.([]interface{})
	return ok, nil
}

func testSameAs(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	return compareEqual(v, args[0]), nil
}

func testDivisibleBy(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	num := toInt(v)
	div := toInt(args[0])
	if div == 0 {
		return false, nil
	}
	return num%div == 0, nil
}

func testEven(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	return toInt(v)%2 == 0, nil
}

func testOdd(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	return toInt(v)%2 != 0, nil
}

func testLower(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	s, ok := v.(string)
	if !ok {
		return false, nil
	}
	return s == strings.ToLower(s) && s != strings.ToUpper(s), nil
}

func testUpper(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	s, ok := v.(string)
	if !ok {
		return false, nil
	}
	return s == strings.ToUpper(s) && s != strings.ToLower(s), nil
}

func testStartsWith(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	s := stringify(v)
	return strings.HasPrefix(s, stringify(args[0])), nil
}

func testEndsWith(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	s := stringify(v)
	return strings.HasSuffix(s, stringify(args[0])), nil
}

func testIn(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	return contains(v, args[0]), nil
}

func testEqualTo(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	return compareEqual(v, args[0]), nil
}

func testGt(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	return compareValues(v, args[0]) > 0, nil
}

func testGte(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	return compareValues(v, args[0]) >= 0, nil
}

func testLt(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	return compareValues(v, args[0]) < 0, nil
}

func testLte(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	return compareValues(v, args[0]) <= 0, nil
}

func testAll(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	items := toSlice(v)
	if items == nil {
		return false, nil
	}
	for _, item := range items {
		if !isTruthy(item) {
			return false, nil
		}
	}
	return true, nil
}

func testAny(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	items := toSlice(v)
	if items == nil {
		return false, nil
	}
	for _, item := range items {
		if isTruthy(item) {
			return true, nil
		}
	}
	return false, nil
}

func testFilter(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	filterName := stringify(args[0])
	if ctx == nil || ctx.env == nil {
		return false, nil
	}
	_, ok := ctx.env.GetFilter(filterName)
	return ok, nil
}

func testBoolean(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	_, ok := v.(bool)
	return ok, nil
}

func testFalse(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	b, ok := v.(bool)
	return ok && !b, nil
}

func testNamespace(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	// Always true if v is not nil and not a basic type
	if v == nil {
		return false, nil
	}
	switch v.(type) {
	case string, bool, int, int64, float64, nil:
		return false, nil
	}
	return true, nil
}

func testEscaped(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	return IsSafe(v), nil
}

func testContainment(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	return contains(args[0], v), nil
}

func testMatching(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	// Regex matching (simplified)
	return false, nil
}

func testSubset(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	if len(args) < 1 {
		return false, nil
	}
	haystack := toSlice(args[0])
	needles := toSlice(v)
	if haystack == nil || needles == nil {
		return false, nil
	}
outer:
	for _, needle := range needles {
		for _, item := range haystack {
			if compareEqual(needle, item) {
				continue outer
			}
		}
		return false, nil
	}
	return true, nil
}

func testSuperset(ctx *ExecutionContext, v interface{}, args ...interface{}) (bool, error) {
	return testSubset(ctx, args[0], []interface{}{v})
}
