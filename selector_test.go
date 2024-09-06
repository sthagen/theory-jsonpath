package jsonpath

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectorInterface(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		tok  any
	}{
		{"name", Name("hi")},
		{"index", Index(42)},
		{"slice", Slice()},
		{"wildcard", Wildcard},
		{"filter", &Filter{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Implements((*Selector)(nil), tc.tok)
		})
	}
}

func TestSelectorString(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		tok  Selector
		str  string
		sing bool
	}{
		{
			name: "name",
			tok:  Name("hi"),
			str:  `"hi"`,
			sing: true,
		},
		{
			name: "name_space",
			tok:  Name("hi there"),
			str:  `"hi there"`,
			sing: true,
		},
		{
			name: "name_quote",
			tok:  Name(`hi "there"`),
			str:  `"hi \"there\""`,
			sing: true,
		},
		{
			name: "name_unicode",
			tok:  Name(`hi 😀`),
			str:  `"hi 😀"`,
			sing: true,
		},
		{
			name: "name_digits",
			tok:  Name(`42`),
			str:  `"42"`,
			sing: true,
		},
		{
			name: "index",
			tok:  Index(42),
			str:  "42",
			sing: true,
		},
		{
			name: "index_big",
			tok:  Index(math.MaxUint32),
			str:  "4294967295",
			sing: true,
		},
		{
			name: "index_zero",
			tok:  Index(0),
			str:  "0",
			sing: true,
		},
		{
			name: "slice_0_4",
			tok:  Slice(0, 4),
			str:  ":4",
		},
		{
			name: "slice_4_5",
			tok:  Slice(4, 5),
			str:  "4:5",
		},
		{
			name: "slice_end_42",
			tok:  Slice(nil, 42),
			str:  ":42",
		},
		{
			name: "slice_start_4",
			tok:  Slice(4),
			str:  "4:",
		},
		{
			name: "slice_start_end_step",
			tok:  Slice(4, 7, 2),
			str:  "4:7:2",
		},
		{
			name: "slice_start_step",
			tok:  Slice(4, nil, 2),
			str:  "4::2",
		},
		{
			name: "slice_end_step",
			tok:  Slice(nil, 4, 2),
			str:  ":4:2",
		},
		{
			name: "slice_step",
			tok:  Slice(nil, nil, 3),
			str:  "::3",
		},
		{
			name: "wildcard",
			tok:  Wildcard,
			str:  "*",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Equal(tc.sing, tc.tok.isSingular())
			buf := new(strings.Builder)
			tc.tok.writeTo(buf)
			a.Equal(tc.str, buf.String())
			a.Equal(tc.str, tc.tok.String())
		})
	}
}

func TestSliceBounds(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	json := []any{"a", "b", "c", "d", "e", "f", "g"}

	extract := func(s SliceSelector) []any {
		lower, upper := s.bounds(len(json))
		res := make([]any, 0, len(json))
		switch {
		case s.step > 0:
			for i := lower; i < upper; i += s.step {
				res = append(res, json[i])
			}
		case s.step < 0:
			for i := upper; lower < i; i += s.step {
				res = append(res, json[i])
			}
		}
		return res
	}

	type lenCase struct {
		length int
		lower  int
		upper  int
	}

	for _, tc := range []struct {
		name  string
		slice SliceSelector
		cases []lenCase
		exp   []any
	}{
		{
			name:  "defaults",
			slice: Slice(),
			exp:   json,
			cases: []lenCase{
				{10, 0, 10},
				{3, 0, 3},
				{99, 0, 99},
			},
		},
		{
			name:  "step_0",
			slice: Slice(nil, nil, 0),
			exp:   []any{},
			cases: []lenCase{
				{10, 0, 0},
				{3, 0, 0},
				{99, 0, 0},
			},
		},
		{
			name:  "3_8_2",
			slice: Slice(3, 8, 2),
			exp:   []any{"d", "f"},
			cases: []lenCase{
				{10, 3, 8},
				{3, 3, 3},
				{99, 3, 8},
			},
		},
		{
			name:  "1_3_1",
			slice: Slice(1, 3, 1),
			exp:   []any{"b", "c"},
			cases: []lenCase{
				{10, 1, 3},
				{2, 1, 2},
				{99, 1, 3},
			},
		},
		{
			name:  "5_defaults",
			slice: Slice(5),
			exp:   []any{"f", "g"},
			cases: []lenCase{
				{10, 5, 10},
				{8, 5, 8},
				{99, 5, 99},
			},
		},
		{
			name:  "1_5_2",
			slice: Slice(1, 5, 2),
			exp:   []any{"b", "d"},
			cases: []lenCase{
				{10, 1, 5},
				{4, 1, 4},
				{99, 1, 5},
			},
		},
		{
			name:  "5_1_neg2",
			slice: Slice(5, 1, -2),
			exp:   []any{"f", "d"},
			cases: []lenCase{
				{10, 1, 5},
				{4, 1, 3},
				{99, 1, 5},
			},
		},
		{
			name:  "def_def_neg1",
			slice: Slice(nil, nil, -1),
			exp:   []any{"g", "f", "e", "d", "c", "b", "a"},
			cases: []lenCase{
				{10, -1, 9},
				{4, -1, 3},
				{99, -1, 98},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.False(tc.slice.isSingular())
			for _, lc := range tc.cases {
				lower, upper := tc.slice.bounds(lc.length)
				a.Equal(lc.lower, lower)
				a.Equal(lc.upper, upper)
			}
			a.Equal(tc.exp, extract(tc.slice))
			a.Equal(tc.slice.start, tc.slice.Start())
			a.Equal(tc.slice.end, tc.slice.End())
			a.Equal(tc.slice.step, tc.slice.Step())
		})
	}
}

func TestSlicePanic(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	a.PanicsWithValue(
		"First value passed to NewSlice is not an integer",
		func() { Slice("hi") },
	)
	a.PanicsWithValue(
		"Second value passed to NewSlice is not an integer",
		func() { Slice(nil, "hi") },
	)
	a.PanicsWithValue(
		"Third value passed to NewSlice is not an integer",
		func() { Slice(nil, 42, "hi") },
	)
}

func TestNameSelect(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		sel  Name
		src  any
		exp  []any
	}{
		{
			name: "got_name",
			sel:  Name("hi"),
			src:  map[string]any{"hi": 42},
			exp:  []any{42},
		},
		{
			name: "got_name_array",
			sel:  Name("hi"),
			src:  map[string]any{"hi": []any{42, true}},
			exp:  []any{[]any{42, true}},
		},
		{
			name: "no_name",
			sel:  Name("hi"),
			src:  map[string]any{"oy": []any{42, true}},
			exp:  []any{},
		},
		{
			name: "src_array",
			sel:  Name("hi"),
			src:  []any{42, true},
			exp:  []any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Equal(tc.exp, tc.sel.Select(tc.src, nil))
		})
	}
}

func TestIndexSelect(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		sel  Index
		src  any
		exp  []any
	}{
		{
			name: "index_zero",
			sel:  Index(0),
			src:  []any{42, true, "hi"},
			exp:  []any{42},
		},
		{
			name: "index_two",
			sel:  Index(2),
			src:  []any{42, true, "hi"},
			exp:  []any{"hi"},
		},
		{
			name: "index_neg_one",
			sel:  Index(-1),
			src:  []any{42, true, "hi"},
			exp:  []any{"hi"},
		},
		{
			name: "index_neg_two",
			sel:  Index(-2),
			src:  []any{42, true, "hi"},
			exp:  []any{true},
		},
		{
			name: "out_of_range",
			sel:  Index(4),
			src:  []any{42, true, "hi"},
			exp:  []any{},
		},
		{
			name: "neg_out_of_range",
			sel:  Index(-4),
			src:  []any{42, true, "hi"},
			exp:  []any{},
		},
		{
			name: "src_object",
			sel:  Index(0),
			src:  map[string]any{"hi": 42},
			exp:  []any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Equal(tc.exp, tc.sel.Select(tc.src, nil))
		})
	}
}

func TestWildcardSelect(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		src  any
		exp  []any
	}{
		{
			name: "object",
			src:  map[string]any{"x": true, "y": []any{true}},
			exp:  []any{true, []any{true}},
		},
		{
			name: "array",
			src:  []any{true, 42, map[string]any{"x": 6}},
			exp:  []any{true, 42, map[string]any{"x": 6}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, ok := tc.src.(map[string]any); ok {
				a.ElementsMatch(tc.exp, Wildcard.Select(tc.src, nil))
			} else {
				a.Equal(tc.exp, Wildcard.Select(tc.src, nil))
			}
		})
	}
}

func TestSliceSelect(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name string
		sel  SliceSelector
		src  any
		exp  []any
	}{
		{
			name: "0_2",
			sel:  Slice(0, 2),
			src:  []any{42, true, "hi"},
			exp:  []any{42, true},
		},
		{
			name: "0_1",
			sel:  Slice(0, 1),
			src:  []any{[]any{42, false}, true, "hi"},
			exp:  []any{[]any{42, false}},
		},
		{
			name: "2_5",
			sel:  Slice(2, 5),
			src:  []any{[]any{42, false}, true, "hi", 98.6, 73, "hi", 22},
			exp:  []any{"hi", 98.6, 73},
		},
		{
			name: "2_5_over_len",
			sel:  Slice(2, 5),
			src:  []any{"x", true, "y"},
			exp:  []any{"y"},
		},
		{
			name: "defaults",
			sel:  Slice(),
			src:  []any{"x", nil, "y", 42},
			exp:  []any{"x", nil, "y", 42},
		},
		{
			name: "default_start",
			sel:  Slice(nil, 3),
			src:  []any{"x", nil, "y", 42, 98.6, 54},
			exp:  []any{"x", nil, "y"},
		},
		{
			name: "default_end",
			sel:  Slice(2),
			src:  []any{"x", true, "y", 42, 98.6, 54},
			exp:  []any{"y", 42, 98.6, 54},
		},
		{
			name: "step_2",
			sel:  Slice(nil, nil, 2),
			src:  []any{"x", true, "y", 42, 98.6, 54},
			exp:  []any{"x", "y", 98.6},
		},
		{
			name: "step_3",
			sel:  Slice(nil, nil, 3),
			src:  []any{"x", true, "y", 42, 98.6, 54, 98, 73},
			exp:  []any{"x", 42, 98},
		},
		{
			name: "negative_step",
			sel:  Slice(nil, nil, -1),
			src:  []any{"x", true, "y", []any{1, 2}},
			exp:  []any{[]any{1, 2}, "y", true, "x"},
		},
		{
			name: "5_0_neg2",
			sel:  Slice(5, 0, -2),
			src:  []any{"x", true, "y", 8, 13, 25, 23, 78, 13},
			exp:  []any{25, 8, true},
		},
		{
			name: "src_object",
			sel:  Slice(0, 2),
			src:  map[string]any{"hi": 42},
			exp:  []any{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a.Equal(tc.exp, tc.sel.Select(tc.src, nil))
		})
	}
}

func TestFilterSelector(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	for _, tc := range []struct {
		name    string
		filter  *Filter
		root    any
		current any
		exp     []any
		str     string
		rand    bool
	}{
		{
			name:   "no_filter",
			filter: &Filter{LogicalOrExpr{}},
			exp:    []any{},
			str:    "",
		},
		{
			name: "array_root",
			filter: &Filter{LogicalOrExpr([]LogicalAndExpr{LogicalAndExpr([]BasicExpr{&ExistExpr{
				&Query{segments: []*Segment{Child(Index(0))}, root: true},
			}})})},
			root:    []any{42, true, "hi"},
			current: map[string]any{"x": 2},
			exp:     []any{2},
			str:     `$[0]`,
		},
		{
			name: "array_root_false",
			filter: &Filter{LogicalOrExpr([]LogicalAndExpr{LogicalAndExpr([]BasicExpr{&ExistExpr{
				&Query{segments: []*Segment{Child(Index(4))}, root: true},
			}})})},
			root:    []any{42, true, "hi"},
			current: map[string]any{"x": 2},
			exp:     []any{},
			str:     `$[4]`,
		},
		{
			name: "object_root",
			filter: &Filter{LogicalOrExpr([]LogicalAndExpr{LogicalAndExpr([]BasicExpr{&ExistExpr{
				&Query{segments: []*Segment{Child(Name("y"))}, root: true},
			}})})},
			root:    map[string]any{"x": 42, "y": "hi"},
			current: map[string]any{"a": 2, "b": 3},
			exp:     []any{2, 3},
			str:     `$["y"]`,
			rand:    true,
		},
		{
			name: "object_root_false",
			filter: &Filter{LogicalOrExpr([]LogicalAndExpr{LogicalAndExpr([]BasicExpr{&ExistExpr{
				&Query{segments: []*Segment{Child(Name("z"))}, root: true},
			}})})},
			root:    map[string]any{"x": 42, "y": "hi"},
			current: map[string]any{"a": 2, "b": 3},
			exp:     []any{},
			str:     `$["z"]`,
			rand:    true,
		},
		{
			name: "array_current",
			filter: &Filter{LogicalOrExpr([]LogicalAndExpr{LogicalAndExpr([]BasicExpr{&ExistExpr{
				&Query{segments: []*Segment{Child(Index(0))}},
			}})})},
			current: []any{[]any{42}},
			exp:     []any{[]any{42}},
			str:     `@[0]`,
		},
		{
			name: "array_current_false",
			filter: &Filter{LogicalOrExpr([]LogicalAndExpr{LogicalAndExpr([]BasicExpr{&ExistExpr{
				&Query{segments: []*Segment{Child(Index(1))}},
			}})})},
			current: []any{[]any{42}},
			exp:     []any{},
			str:     `@[1]`,
		},
		{
			name: "object_current",
			filter: &Filter{LogicalOrExpr([]LogicalAndExpr{LogicalAndExpr([]BasicExpr{&ExistExpr{
				&Query{segments: []*Segment{Child(Name("x"))}},
			}})})},
			current: []any{map[string]any{"x": 42}},
			exp:     []any{map[string]any{"x": 42}},
			str:     `@["x"]`,
		},
		{
			name: "object_current_false",
			filter: &Filter{LogicalOrExpr([]LogicalAndExpr{LogicalAndExpr([]BasicExpr{&ExistExpr{
				&Query{segments: []*Segment{Child(Name("y"))}},
			}})})},
			current: []any{map[string]any{"x": 42}},
			exp:     []any{},
			str:     `@["y"]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.rand {
				a.ElementsMatch(tc.exp, tc.filter.Select(tc.current, tc.root))
			} else {
				a.Equal(tc.exp, tc.filter.Select(tc.current, tc.root))
			}
			a.Equal(tc.str, tc.filter.String())
			a.Equal(tc.str, bufString(tc.filter))
			a.False(tc.filter.isSingular())
		})
	}
}
