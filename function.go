package jsonpath

//go:generate stringer -linecomment -output function_string.go -type LogicalType,PathType,FuncType

import (
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"
	"strings"
	"sync"
	"unicode/utf8"
)

// PathType represents a path type.
type PathType uint8

//revive:disable:exported
const (
	// A type containing a single value.
	PathValue PathType = iota + 1 // ValueType

	// A boolean type.
	PathLogical // LogicalType

	// A type containing a list of nodes.
	PathNodes // NodesType
)

// FuncType defines the different types of function arguments and return
// values. Function extensions require arguments that convert to specific
// PathValues and define return values in terms of these types.
type FuncType uint8

const (
	// A literal JSON value.
	FuncLiteral FuncType = iota + 1 // FuncLiteral

	// A value from a singular query.
	FuncSingularQuery // FuncSingularQuery

	// A JSON value, used to represent functions that return [ValueType].
	FuncValue // FuncValue

	// A node list, either from a filter query argument, or a function that
	// returns [NodesType].
	FuncNodeList // FuncNodeList

	// A logical, either from a logical expression, or from a function that
	// returns [LogicalType].
	FuncLogical // FuncLogical
)

// convertsTo returns true if a function argument of type ft can be converted
// to pv.
func (ft FuncType) convertsTo(pv PathType) bool {
	switch ft {
	case FuncLiteral, FuncValue:
		return pv == PathValue
	case FuncSingularQuery:
		return true
	case FuncNodeList:
		return pv != PathValue
	case FuncLogical:
		return pv == PathLogical
	default:
		return false
	}
}

// JSONPathValue defines the interface for JSON path values.
type JSONPathValue interface {
	stringWriter
	// PathType returns the JSONPathValue's PathType.
	PathType() PathType
	// FuncType returns the JSONPathValue's FuncType.
	FuncType() FuncType
}

// NodesType defines the JSONPath type representing a node list; in other
// words, a list of JSON values.
type NodesType []any

// PathType returns PathNodes. Defined by the JSONPathValue interface.
func (NodesType) PathType() PathType { return PathNodes }

// FuncType returns FuncNodeList. Defined by the JSONPathValue interface.
func (NodesType) FuncType() FuncType { return FuncNodeList }

// newNodesTypeFrom attempts to convert value to a NodesType.
func newNodesTypeFrom(value JSONPathValue) NodesType {
	switch v := value.(type) {
	case NodesType:
		return v
	case *ValueType:
		return NodesType([]any{v.any})
	case nil:
		return NodesType([]any{})
	default:
		panic(fmt.Sprintf("unexpected argument of type %T", v))
	}
}

// writeTo writes a string representation of the NodesType to buf.
func (NodesType) writeTo(buf *strings.Builder) {
	buf.WriteString("NodesType")
}

// LogicalType is a JSONPath type that represents true or false.
type LogicalType uint8

//revive:disable:exported
const (
	LogicalFalse LogicalType = iota // false
	LogicalTrue                     // true
)

// logicalFrom converts b to a LogicalType.
func logicalFrom(b bool) LogicalType {
	if b {
		return LogicalTrue
	}
	return LogicalFalse
}

// Bool returns the boolean equivalent to lt.
func (lt LogicalType) Bool() bool { return lt == LogicalTrue }

// PathType returns PathLogical. Defined by the JSONPathValue interface.
func (LogicalType) PathType() PathType { return PathLogical }

// FuncType returns FuncLogical. Defined by the JSONPathValue interface.
func (LogicalType) FuncType() FuncType { return FuncLogical }

// newNodesTypeFrom attempts to convert value to a NodesType.
func newLogicalTypeFrom(value JSONPathValue) LogicalType {
	switch v := value.(type) {
	case LogicalType:
		return v
	case NodesType:
		return logicalFrom(len(v) > 0)
	case nil:
		return LogicalFalse
	default:
		panic(fmt.Sprintf("unexpected argument of type %T", v))
	}
}

// writeTo writes a string representation of lt to buf.
func (lt LogicalType) writeTo(buf *strings.Builder) {
	buf.WriteString(lt.String())
}

// ValueType encapsulates a JSON value, which should be a string, integer,
// float, nil, true, false, []any, or map[string]any. A nil ValueType pointer
// indicates no value.
type ValueType struct {
	any
}

// PathType returns PathValue. Defined by the JSONPathValue interface.
func (*ValueType) PathType() PathType { return PathValue }

// FuncType returns FuncValue. Defined by the JSONPathValue interface.
func (*ValueType) FuncType() FuncType { return FuncValue }

// newValueTypeFrom attempts to convert value to a ValueType.
func newValueTypeFrom(value JSONPathValue) *ValueType {
	switch v := value.(type) {
	case *ValueType:
		return v
	case nil:
		return nil
	}
	panic(fmt.Sprintf("unexpected argument of type %T", value))
}

// Returns true if vt.any is truthy.
func (vt *ValueType) testFilter(_, _ any) bool {
	switch v := vt.any.(type) {
	case nil:
		return false
	case bool:
		return v
	case int:
		return v != 0
	case int8:
		return v != int8(0)
	case int16:
		return v != int16(0)
	case int32:
		return v != int32(0)
	case int64:
		return v != int64(0)
	case uint:
		return v != 0
	case uint8:
		return v != uint8(0)
	case uint16:
		return v != uint16(0)
	case uint32:
		return v != uint32(0)
	case uint64:
		return v != uint64(0)
	case float32:
		return v != float32(0)
	case float64:
		return v != float64(0)
	default:
		return true
	}
}

// writeTo writes a string representation of vt to buf.
func (vt *ValueType) writeTo(buf *strings.Builder) {
	buf.WriteString("ValueType")
}

//nolint:gochecknoglobals
var (
	registryMu sync.RWMutex
	registry   = make(map[string]*Function)
)

// Register registers a function extension by its name. If fn is nil or
// Register is called twice with the same fn.name, it panics.
func Register(fn *Function) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if fn == nil {
		panic("jsonpath: Register function is nil")
	}
	if _, dup := registry[fn.Name]; dup {
		panic("jsonpath: Register called twice for function " + fn.Name)
	}
	registry[fn.Name] = fn
}

// GetFunction returns a reference to the registered function named name.
// Returns nil if no function with that name has been registered.
func GetFunction(name string) *Function {
	registryMu.RLock()
	defer registryMu.RUnlock()
	function := registry[name]
	return function
}

// registerFunctions registers the functions defined by [RFC 9535].
func registerFunctions() {
	Register(&Function{
		Name:       "length",
		ResultType: FuncValue,
		Validate:   checkLengthArgs,
		Evaluate:   lengthFunc,
	})
	Register(&Function{
		Name:       "count",
		ResultType: FuncValue,
		Validate:   checkCountArgs,
		Evaluate:   countFunc,
	})
	Register(&Function{
		Name:       "value",
		ResultType: FuncValue,
		Validate:   checkValueArgs,
		Evaluate:   valueFunc,
	})
	Register(&Function{
		Name:       "match",
		ResultType: FuncLogical,
		Validate:   checkMatchArgs,
		Evaluate:   matchFunc,
	})
	Register(&Function{
		Name:       "search",
		ResultType: FuncLogical,
		Validate:   checkSearchArgs,
		Evaluate:   searchFunc,
	})
}

//nolint:gochecknoinits
func init() { registerFunctions() }

// checkLengthArgs checks the argument expressions to length() and returns an
// error if there is not exactly one expression that results in a
// [PathValue]-compatible value.
//
//nolint:err113
func checkLengthArgs(fea []FunctionExprArg) error {
	if len(fea) != 1 {
		return fmt.Errorf("expected 1 argument but found %v", len(fea))
	}

	kind := fea[0].asTypeKind()
	if !kind.convertsTo(PathValue) {
		return errors.New("cannot convert length() argument to ValueType")
	}

	return nil
}

// lengthFunc extracts the single argument passed in jv and returns its
// length. Panics if jv[0] doesn't exist or is not convertible to [ValueType].
//
//   - if jv[0] is nil, the result is nil
//   - If jv[0] is a string, the result is the number of Unicode scalar values
//     in the string.
//   - If jv[0] is a []any, the result is the number of elements in the slice.
//   - If jv[0] is an map[string]any, the result is the number of members in
//     the map.
//   - For any other value, the result is nil.
//
//nolint:ireturn
func lengthFunc(jv []JSONPathValue) JSONPathValue {
	v := newValueTypeFrom(jv[0])
	if v == nil {
		return nil
	}
	switch v := v.any.(type) {
	case string:
		// Unicode scalar values
		return &ValueType{utf8.RuneCountInString(v)}
	case []any:
		return &ValueType{len(v)}
	case map[string]any:
		return &ValueType{len(v)}
	default:
		return nil
	}
}

// checkCountArgs checks the argument expressions to count() and returns an
// error if there is not exactly one expression that results in a
// [PathNodes]-compatible value.
//
//nolint:err113
func checkCountArgs(fea []FunctionExprArg) error {
	if len(fea) != 1 {
		return fmt.Errorf("expected 1 argument but found %v", len(fea))
	}

	kind := fea[0].asTypeKind()
	if !kind.convertsTo(PathNodes) {
		return errors.New("cannot convert count() argument to PathNodes")
	}

	return nil
}

// countFunc implements the [RFC 9535]-standard count function. The result is
// a ValueType containing an unsigned integer for the number of nodes
// in jv[0]. Panics if jv[0] doesn't exist or is not convertible to
// [NodesType].
//
//nolint:ireturn
func countFunc(jv []JSONPathValue) JSONPathValue {
	return &ValueType{len(newNodesTypeFrom(jv[0]))}
}

// checkValueArgs checks the argument expressions to value() and returns an
// error if there is not exactly one expression that results in a
// [PathNodes]-compatible value.
//
//nolint:err113
func checkValueArgs(fea []FunctionExprArg) error {
	if len(fea) != 1 {
		return fmt.Errorf("expected 1 argument but found %v", len(fea))
	}

	kind := fea[0].asTypeKind()
	if !kind.convertsTo(PathNodes) {
		return errors.New("cannot convert value() argument to PathNodes")
	}

	return nil
}

// valueFunc implements the [RFC 9535]-standard value function. Panics if
// jv[0] doesn't exist or is not convertible to [NodesType]. Otherwise:
//
//   - If jv[0] contains a single node, the result is the value of the node.
//   - If jv[0] is empty or contains multiple nodes, the result is nil.
//
//nolint:ireturn
func valueFunc(jv []JSONPathValue) JSONPathValue {
	nodes := newNodesTypeFrom(jv[0])
	if len(nodes) == 1 {
		return &ValueType{nodes[0]}
	}
	return nil
}

// checkMatchArgs checks the argument expressions to match() and returns an
// error if there are not exactly two expressions that result in
// [PathValue]-compatible values.
//
//nolint:err113
func checkMatchArgs(fea []FunctionExprArg) error {
	const matchArgLen = 2
	if len(fea) != matchArgLen {
		return fmt.Errorf("expected 2 arguments but found %v", len(fea))
	}

	for i, arg := range fea {
		kind := arg.asTypeKind()
		if !kind.convertsTo(PathValue) {
			return fmt.Errorf("cannot convert match() argument %v to PathNodes", i+1)
		}
	}

	return nil
}

// matchFunc implements the [RFC 9535]-standard match function. If jv[0] and
// jv[1] evaluate to strings, the second is compiled into a regular expression with
// implied \A and \z anchors and used to match the first, returning LogicalTrue for
// a match and LogicalFalse for no match. Returns LogicalFalse if either jv value
// is not a string or if jv[1] fails to compile.
//
//nolint:ireturn
func matchFunc(jv []JSONPathValue) JSONPathValue {
	if v, ok := newValueTypeFrom(jv[0]).any.(string); ok {
		if r, ok := newValueTypeFrom(jv[1]).any.(string); ok {
			if rc := compileRegex(`\A` + r + `\z`); rc != nil {
				return logicalFrom(rc.MatchString(v))
			}
		}
	}
	return LogicalFalse
}

// checkSearchArgs checks the argument expressions to search() and returns an
// error if there are not exactly two expressions that result in
// [PathValue]-compatible values.
//
//nolint:err113
func checkSearchArgs(fea []FunctionExprArg) error {
	const searchArgLen = 2
	if len(fea) != searchArgLen {
		return fmt.Errorf("expected 2 arguments but found %v", len(fea))
	}

	for i, arg := range fea {
		kind := arg.asTypeKind()
		if !kind.convertsTo(PathValue) {
			return fmt.Errorf("cannot convert search() argument %v to PathNodes", i+1)
		}
	}

	return nil
}

// searchFunc implements the [RFC 9535]-standard search function. If both jv[0]
// and jv[1] contain strings, the latter is compiled into a regular expression and used
// to match the former, returning LogicalTrue for a match and LogicalFalse for no
// match. Returns LogicalFalse if either value is not a string, or if jv[1]
// fails to compile.
//
//nolint:ireturn
func searchFunc(jv []JSONPathValue) JSONPathValue {
	if val, ok := newValueTypeFrom(jv[0]).any.(string); ok {
		if r, ok := newValueTypeFrom(jv[1]).any.(string); ok {
			if rc := compileRegex(r); rc != nil {
				return logicalFrom(rc.MatchString(val))
			}
		}
	}
	return LogicalFalse
}

// compileRegex compiles str into a regular expression or returns an error. To
// comply with RFC 9485 regular expression semantics, all instances of "." are
// replaced with "[^\n\r]". This sadly requires compiling the regex twice:
// once to produce an AST to replace "." nodes, and a second time for the
// final regex.
func compileRegex(str string) *regexp.Regexp {
	// First compile AST and replace "." with [^\n\r].
	// https://www.rfc-editor.org/rfc/rfc9485.html#name-pcre-re2-and-ruby-regexps
	r, err := syntax.Parse(str, syntax.Perl|syntax.DotNL)
	if err != nil {
		// Could use some way to log these errors rather than failing silently.
		return nil
	}

	replaceDot(r)
	re, _ := regexp.Compile(r.String())
	return re
}

//nolint:gochecknoglobals
var clrf, _ = syntax.Parse("[^\n\r]", syntax.Perl)

// replaceDot recurses re to replace all "." nodes with "[^\n\r]" nodes.
func replaceDot(re *syntax.Regexp) {
	if re.Op == syntax.OpAnyChar {
		*re = *clrf
	} else {
		for _, re := range re.Sub {
			replaceDot(re)
		}
	}
}

// Function defines a JSONPath function. Use [Register] to register a new
// function.
type Function struct {
	// Name is the name of the function. Must be unique among all functions.
	Name string

	// ResultType defines the type of the function return value.
	ResultType FuncType

	// Validate executes at parse time to validate that all the args to
	// the function are compatible with the function.
	Validate func(args []FunctionExprArg) error

	// Evaluate executes the function against args and returns the result of
	// type ResultType.
	Evaluate func(args []JSONPathValue) JSONPathValue
}

// FunctionExprArg defines the interface for function argument expressions.
type FunctionExprArg interface {
	stringWriter
	// evaluate evaluates the function expression against current and root and
	// returns the resulting JSONPathValue.
	execute(current, root any) JSONPathValue
	// asTypeKind returns the FuncType that defines the type of the return
	// value of JSONPathValue.
	asTypeKind() FuncType
}

// CompVal defines the interface for comparable values in filter
// expressions.
type CompVal interface {
	stringWriter
	// asValue returns the value to be compared.
	asValue(current, root any) JSONPathValue
}

// literalArg represents a literal JSON value, excluding objects and arrays.
type literalArg struct {
	// Number, string, bool, or null
	literal any
}

// execute returns a [ValueType] containing the literal value. Defined by the
// [FunctionExprArg] interface.
//
//nolint:ireturn
func (la *literalArg) execute(_, _ any) JSONPathValue {
	return &ValueType{la.literal}
}

// asTypeKind returns FuncLiteral. Defined by the [FunctionExprArg] interface.
func (la *literalArg) asTypeKind() FuncType {
	return FuncLiteral
}

// writeTo writes a string representation of la to buf.
func (la *literalArg) writeTo(buf *strings.Builder) {
	if la.literal == nil {
		buf.WriteString("null")
	} else {
		fmt.Fprintf(buf, "%#v", la.literal)
	}
}

// asValue returns la.literal as a [ValueType]. Defined by the [comparableVal]
// interface.
//
//nolint:ireturn
func (la *literalArg) asValue(_, _ any) JSONPathValue {
	return &ValueType{la.literal}
}

// singularQuery represents a query that produces a single node (JSON value),
// or nothing.
type singularQuery struct {
	// The kind of singular query, relative (from the current node) or
	// absolute (from the root node).
	relative bool
	// The query Name and/or Index selectors.
	selectors []Selector
}

// execute returns a [ValueType] containing the return value of executing sq.
// Defined by the [FunctionExprArg] interface.
//
//nolint:ireturn
func (sq *singularQuery) execute(current, root any) JSONPathValue {
	target := root
	if sq.relative {
		target = current
	}

	for _, seg := range sq.selectors {
		res := seg.Select(target, nil)
		if len(res) == 0 {
			return nil
		}
		target = res[0]
	}

	return &ValueType{target}
}

// asTypeKind returns FuncSingularQuery. Defined by the [FunctionExprArg]
// interface.
func (*singularQuery) asTypeKind() FuncType {
	return FuncSingularQuery
}

// asValue returns the result of executing sq.execute against current and root.
// Defined by the [comparableVal] interface.
//
//nolint:ireturn
func (sq *singularQuery) asValue(current, root any) JSONPathValue {
	return sq.execute(current, root)
}

// writeTo writes a string representation of sq to buf.
func (sq *singularQuery) writeTo(buf *strings.Builder) {
	if sq.relative {
		buf.WriteRune('@')
	} else {
		buf.WriteRune('$')
	}

	for _, seg := range sq.selectors {
		buf.WriteRune('[')
		seg.writeTo(buf)
		buf.WriteRune(']')
	}
}

// filterQuery represents a JSONPath Query used in a filter expression.
type filterQuery struct {
	*Query
}

// execute returns a [NodesType] containing the result of executing fq.
// Defined by the [FunctionExprArg] interface.
//
//nolint:ireturn
func (fq *filterQuery) execute(current, root any) JSONPathValue {
	return NodesType(fq.Select(current, root))
}

// asTypeKind returns FuncSingularQuery if fq is a singular query, and
// FuncNodeList if it is not. Defined by the [FunctionExprArg] interface.
func (fq *filterQuery) asTypeKind() FuncType {
	if fq.isSingular() {
		return FuncSingularQuery
	}
	return FuncNodeList
}

// writeTo writes a string representation of fq to buf.
func (fq *filterQuery) writeTo(buf *strings.Builder) {
	buf.WriteString(fq.Query.String())
}

// FunctionExpr represents a function expression, consisting of a named
// function and its arguments.
type FunctionExpr struct {
	args []FunctionExprArg
	fn   *Function
}

// NewFunctionExpr creates and returns a new FunctionExpr. Returns an error if
// the function is not registered or its args are invalid.
func NewFunctionExpr(name string, args []FunctionExprArg) (*FunctionExpr, error) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if fn, ok := registry[name]; ok {
		if err := fn.Validate(args); err != nil {
			return nil, err
		}
		return &FunctionExpr{args: args, fn: fn}, nil
	}
	return nil, fmt.Errorf("%w: unknown jsonpath function %v()", ErrPathParse, name)
}

// writeTo writes the string representation of fe to buf.
func (fe *FunctionExpr) writeTo(buf *strings.Builder) {
	buf.WriteString(fe.fn.Name + "(")
	for i, arg := range fe.args {
		arg.writeTo(buf)
		if i < len(fe.args)-1 {
			buf.WriteString(", ")
		}
	}
	buf.WriteRune(')')
}

// execute returns a [NodesType] containing the results of executing each
// argument in fe.args. Defined by the [FunctionExprArg] interface.
//
//nolint:ireturn
func (fe *FunctionExpr) execute(current, root any) JSONPathValue {
	res := []JSONPathValue{}
	for _, a := range fe.args {
		res = append(res, a.execute(current, root))
	}

	return fe.fn.Evaluate(res)
}

// asTypeKind returns the result type of the registered function named
// fe.name. Defined by the [FunctionExprArg] interface.
func (fe *FunctionExpr) asTypeKind() FuncType {
	return fe.fn.ResultType
}

// asValue returns the result of executing fe.execute against current and root.
// Defined by the [comparableVal] interface.
//
//nolint:ireturn
func (fe *FunctionExpr) asValue(current, root any) JSONPathValue {
	return fe.execute(current, root)
}

// testFilter executes fe and returns true if the function returns a truthy
// value:
//
//   - If the result is [NodesType], returns true if it is not empty.
//   - If the result is [*ValueType], returns true if its underlying value
//     is truthy.
//   - If the result is [LogicalType], returns the underlying boolean.
//
// Returns false in all other cases.
func (fe *FunctionExpr) testFilter(current, root any) bool {
	switch res := fe.execute(current, root).(type) {
	case NodesType:
		return len(res) > 0
	case *ValueType:
		return res.testFilter(current, root)
	case LogicalType:
		return res.Bool()
	default:
		return false
	}
}

// NotFuncExpr represents a "!func()" expression. It reverses the result of
// the return value of a function expression.
type NotFuncExpr struct {
	*FunctionExpr
}

// testFilter returns the inverse of nf.FunctionExpr.testFilter().
func (nf NotFuncExpr) testFilter(current, root any) bool {
	return !nf.FunctionExpr.testFilter(current, root)
}
