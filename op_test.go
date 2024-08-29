package jsonpath

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompOp(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		op  CompOp
		str string
	}{
		{EqualTo, "=="},
		{NotEqualTo, "!="},
		{LessThan, "<"},
		{LessThanEqualTo, "<="},
		{GreaterThan, ">"},
		{GreaterThanEqualTo, ">="},
	} {
		a.Equal(tc.str, tc.op.String())
	}
}

func TestEqualTo(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name  string
		left  any
		right any
		exp   bool
	}{
		{"int_zeros", 0, 0, true},
		{"int_ones", 1, 1, true},
		{"int_zero_one", 0, 1, false},
		{"int8_zeros", int8(0), int8(0), true},
		{"int8_ones", int8(1), int8(1), true},
		{"int8_zero_one", int8(0), int8(1), false},
		{"int16_zeros", int16(0), int16(0), true},
		{"int16_ones", int16(1), int16(1), true},
		{"int16_zero_one", int16(0), int16(1), false},
		{"int32_zeros", int32(0), int32(0), true},
		{"int32_ones", int32(1), int32(1), true},
		{"int32_zero_one", int32(0), int32(1), false},
		{"int64_zeros", int64(0), int64(0), true},
		{"int64_ones", int64(1), int64(1), true},
		{"int64_zero_one", int64(0), int64(1), false},
		{"uint_zeros", uint(0), uint(0), true},
		{"uint_ones", uint(1), uint(1), true},
		{"uint_zero_one", uint(0), uint(1), false},
		{"uint8_zeros", uint8(0), uint8(0), true},
		{"uint8_ones", uint8(1), uint8(1), true},
		{"uint8_zero_one", uint8(0), uint8(1), false},
		{"uint16_zeros", uint16(0), uint16(0), true},
		{"uint16_ones", uint16(1), uint16(1), true},
		{"uint16_zero_one", uint16(0), uint16(1), false},
		{"uint32_zeros", uint32(0), uint32(0), true},
		{"uint32_ones", uint32(1), uint32(1), true},
		{"uint32_zero_one", uint32(0), uint32(1), false},
		{"uint64_zeros", uint64(0), uint64(0), true},
		{"uint64_ones", uint64(1), uint64(1), true},
		{"uint64_zero_one", uint64(0), uint64(1), false},
		{"float32_zeros", float32(0), float32(0), true},
		{"float32_ones", float32(1), float32(1), true},
		{"float32_zero_one", float32(0), float32(1), false},
		{"float64_zeros", float64(0), float64(0), true},
		{"float64_ones", float64(1), float64(1), true},
		{"float64_zero_one", float64(0), float64(1), false},
		{"int_float_true", int64(10), float64(10), true},
		{"int_float_false", int64(10), float64(11), false},
		{"empty_strings", "", "", true},
		{"strings", "xyz", "xyz", true},
		{"strings_false", "xyz", "abc", false},
		{"unicode_strings", "foü", "foü", true},
		{"emoji_strings", "hi 😀", "hi 😀", true},
		{"trues", true, true, true},
		{"true_false", true, false, false},
		{"arrays_equal", []any{1, 2, 3}, []any{1, 2, 3}, true},
		{"arrays_ne", []any{1, 2, 3}, []any{1, 2, 3, 4}, false},
		{"nils", nil, nil, true},
		{"nil_not_nil", nil, 2, false},
		{"objects_eq", map[string]any{"x": 1, "y": 2}, map[string]any{"x": 1, "y": 2}, true},
		{"object_keys_ne", map[string]any{"x": 1, "y": 2}, map[string]any{"x": 1, "z": 2}, false},
		{"object_vals_ne", map[string]any{"x": 1, "y": 2}, map[string]any{"x": 1, "y": 3}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Equal(tc.exp, valueEqualTo(tc.left, tc.right))
			a.Equal(tc.exp, equalTo(&ValueType{tc.left}, &ValueType{tc.right}))
		})
	}

	t.Run("not_comparable", func(t *testing.T) {
		t.Parallel()
		a.False(valueEqualTo(42, "x"))
	})
}

func TestLessThan(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name  string
		left  any
		right any
		exp   bool
	}{
		{"int_zeros", 0, 0, false},
		{"int_zero_one", 0, 1, true},
		{"int_one_zero", 1, 0, false},
		{"int8_zeros", 0, 0, false},
		{"int8_zero_one", 0, 1, true},
		{"int8_one_zero", 1, 0, false},
		{"int16_zeros", 0, 0, false},
		{"int16_zero_one", 0, 1, true},
		{"int16_one_zero", 1, 0, false},
		{"int32_zeros", 0, 0, false},
		{"int32_zero_one", 0, 1, true},
		{"int32_one_zero", 1, 0, false},
		{"int64_zeros", 0, 0, false},
		{"int64_zero_one", 0, 1, true},
		{"int64_one_zero", 1, 0, false},
		{"uint_zeros", 0, 0, false},
		{"uint_zero_one", 0, 1, true},
		{"uint_one_zero", 1, 0, false},
		{"uint8_zeros", 0, 0, false},
		{"uint8_zero_one", 0, 1, true},
		{"uint8_one_zero", 1, 0, false},
		{"uint16_zeros", 0, 0, false},
		{"uint16_zero_one", 0, 1, true},
		{"uint16_one_zero", 1, 0, false},
		{"uint32_zeros", 0, 0, false},
		{"uint32_zero_one", 0, 1, true},
		{"uint32_one_zero", 1, 0, false},
		{"uint64_zeros", 0, 0, false},
		{"uint64_zero_one", 0, 1, true},
		{"uint64_one_zero", 1, 0, false},
		{"float32_zeros", 0, 0, false},
		{"float32_zero_one", 0, 1, true},
		{"float32_one_zero", 1, 0, false},
		{"float64_zeros", 0, 0, false},
		{"float64_zero_one", 0, 1, true},
		{"float64_one_zero", 1, 0, false},
		{"int_float_true", 12, 98.6, true},
		{"int_float_false", 99, 98.6, false},
		{"float_int_false", 98.6, 98, false},
		{"float_int_true", 98.6, 99, true},
		{"empty_string_sting", "", "x", true},
		{"empty_strings", "", "", false},
		{"string_a_b", "a", "b", true},
		{"string_c_b", "c", "b", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Equal(tc.exp, valueLessThan(tc.left, tc.right))
			a.Equal(tc.exp, lessThan(&ValueType{tc.left}, &ValueType{tc.right}))
		})
	}

	t.Run("not_comparable", func(t *testing.T) {
		t.Parallel()
		a.False(valueLessThan(42, "x"))
		a.False(valueLessThan([]any{0}, []any{1}))
	})
}

func TestComparisonExpr(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name    string
		left    comparableVal
		right   comparableVal
		root    any
		current any
		expect  []bool
		str     string
	}{
		{
			name:   "literal_numbers_eq",
			left:   &literalArg{42},
			right:  &literalArg{42},
			expect: []bool{true, false, false, false, true, true},
			str:    "42 %v 42",
		},
		{
			name:   "literal_numbers_lt",
			left:   &literalArg{42},
			right:  &literalArg{43},
			expect: []bool{false, true, true, false, true, false},
			str:    "42 %v 43",
		},
		{
			name:   "literal_numbers_gt",
			left:   &literalArg{43},
			right:  &literalArg{42},
			expect: []bool{false, true, false, true, false, true},
			str:    "43 %v 42",
		},
		{
			name:   "literal_strings_eq",
			left:   &literalArg{"x"},
			right:  &literalArg{"x"},
			expect: []bool{true, false, false, false, true, true},
			str:    `"x" %v "x"`,
		},
		{
			name:   "literal_strings_lt",
			left:   &literalArg{"x"},
			right:  &literalArg{"y"},
			expect: []bool{false, true, true, false, true, false},
			str:    `"x" %v "y"`,
		},
		{
			name:   "literal_strings_gt",
			left:   &literalArg{"y"},
			right:  &literalArg{"x"},
			expect: []bool{false, true, false, true, false, true},
			str:    `"y" %v "x"`,
		},
		{
			name:   "query_numbers_eq",
			left:   &singularQuery{selectors: []Selector{Name("x")}},
			right:  &singularQuery{selectors: []Selector{Name("y")}},
			root:   map[string]any{"x": 42, "y": 42},
			expect: []bool{true, false, false, false, true, true},
			str:    `$["x"] %v $["y"]`,
		},
		{
			name:    "query_numbers_lt",
			left:    &singularQuery{selectors: []Selector{Name("x")}, relative: true},
			right:   &singularQuery{selectors: []Selector{Name("y")}, relative: true},
			current: map[string]any{"x": 42, "y": 43},
			expect:  []bool{false, true, true, false, true, false},
			str:     `@["x"] %v @["y"]`,
		},
		{
			name:    "query_string_gt",
			left:    &singularQuery{selectors: []Selector{Name("y")}},
			right:   &singularQuery{selectors: []Selector{Name("x")}},
			current: map[string]any{"x": "x", "y": "y"},
			expect:  []bool{false, true, false, true, false, true},
			str:     `$["y"] %v $["x"]`,
		},
		{
			name: "func_numbers_eq",
			left: &FunctionExpr{
				args: []FunctionExprArg{&singularQuery{selectors: []Selector{Name("x")}}},
				fn:   registry["length"],
			},
			right: &FunctionExpr{
				args: []FunctionExprArg{&singularQuery{selectors: []Selector{Name("y")}}},
				fn:   registry["length"],
			},
			root:   map[string]any{"x": "xx", "y": "yy"},
			expect: []bool{true, false, false, false, true, true},
			str:    `length($["x"]) %v length($["y"])`,
		},
		{
			name: "func_numbers_lt",
			left: &FunctionExpr{
				args: []FunctionExprArg{&singularQuery{selectors: []Selector{Name("x")}}},
				fn:   registry["length"],
			},
			right: &FunctionExpr{
				args: []FunctionExprArg{&singularQuery{selectors: []Selector{Name("y")}}},
				fn:   registry["length"],
			},
			root:   map[string]any{"x": "xx", "y": "yyy"},
			expect: []bool{false, true, true, false, true, false},
			str:    `length($["x"]) %v length($["y"])`,
		},
		{
			name: "func_strings_gt",
			left: &FunctionExpr{
				args: []FunctionExprArg{&filterQuery{NewQuery([]*Segment{Child(Name("y"))})}},
				fn:   registry["value"],
			},
			right: &FunctionExpr{
				args: []FunctionExprArg{&filterQuery{NewQuery([]*Segment{Child(Name("x"))})}},
				fn:   registry["value"],
			},
			root:   map[string]any{"x": "x", "y": "y"},
			expect: []bool{false, true, false, true, false, true},
			str:    `value(@["y"]) %v value(@["x"])`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for i, op := range []struct {
				name string
				op   CompOp
			}{
				{"eq", EqualTo},
				{"ne", NotEqualTo},
				{"lt", LessThan},
				{"gt", GreaterThan},
				{"le", LessThanEqualTo},
				{"ge", GreaterThanEqualTo},
			} {
				t.Run(op.name, func(t *testing.T) {
					t.Parallel()
					cmp := &ComparisonExpr{tc.left, op.op, tc.right}
					a.Equal(tc.expect[i], cmp.testFilter(tc.current, tc.root))
					a.Equal(fmt.Sprintf(tc.str, op.op), bufString(cmp))
				})
			}
		})

		t.Run("unknown_op", func(t *testing.T) {
			t.Parallel()
			cmp := &ComparisonExpr{tc.left, CompOp(16), tc.right}
			a.Equal(fmt.Sprintf(tc.str, cmp.Op), bufString(cmp))
			a.PanicsWithValue("Unknown operator CompOp(16)", func() {
				cmp.testFilter(tc.current, tc.root)
			})
		})
	}
}
