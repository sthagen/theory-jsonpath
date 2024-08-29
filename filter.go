package jsonpath

import (
	"strings"
)

// basicExpr defines the interface for filter expressions.
type basicExpr interface {
	stringWriter
	// testFilter executes the filter expression on current and root and
	// returns true or false depending on the truthiness of its result.
	testFilter(current, root any) bool
}

// LogicalAndExpr represents a list of one or more expressions ANDed together
// by the && operator.
type LogicalAndExpr []basicExpr

// testFilter returns true if all of la's expressions return true.
// Short-circuits and returns false for the first expression that returns
// false.
func (la LogicalAndExpr) testFilter(current, root any) bool {
	for _, e := range la {
		if !e.testFilter(current, root) {
			return false
		}
	}
	return true
}

// writeTo writes the string representation of la to buf.
func (la LogicalAndExpr) writeTo(buf *strings.Builder) {
	for i, e := range la {
		e.writeTo(buf)
		if i < len(la)-1 {
			buf.WriteString(" && ")
		}
	}
}

// LogicalOrExpr represents a list of one or more expressions ORed together by
// the || operator.
type LogicalOrExpr []LogicalAndExpr

func (lo LogicalOrExpr) testFilter(current, root any) bool {
	for _, e := range lo {
		if e.testFilter(current, root) {
			return true
		}
	}
	return false
}

// writeTo writes the string representation of lo to buf.
func (lo LogicalOrExpr) writeTo(buf *strings.Builder) {
	for i, e := range lo {
		e.writeTo(buf)
		if i < len(lo)-1 {
			buf.WriteString(" || ")
		}
	}
}

// execute evaluates lo and returns LogicalTrue when it returns true and
// LogicalFalse when it returns false.
//
//nolint:ireturn
func (lo LogicalOrExpr) execute(current, root any) JSONPathValue {
	return logicalFrom(lo.testFilter(current, root))
}

// asTypeKind returns FuncLogical. Defined by the [FunctionExprArg] interface.
func (lo LogicalOrExpr) asTypeKind() FuncType {
	return FuncLogical
}

// ParenExpr represents a parenthesized expression.
type ParenExpr struct {
	LogicalOrExpr
}

// writeTo writes a string representation of p to buf.
func (p *ParenExpr) writeTo(buf *strings.Builder) {
	buf.WriteRune('(')
	p.LogicalOrExpr.writeTo(buf)
	buf.WriteRune(')')
}

// NotParenExpr represents a parenthesized expression preceded with a !.
type NotParenExpr struct {
	LogicalOrExpr
}

// writeTo writes a string representation of p to buf.
func (np *NotParenExpr) writeTo(buf *strings.Builder) {
	buf.WriteString("!(")
	np.LogicalOrExpr.writeTo(buf)
	buf.WriteRune(')')
}

// testFilter returns false if the np.LogicalOrExpression returns true and
// true if it returns false.
func (np *NotParenExpr) testFilter(current, root any) bool {
	return !np.LogicalOrExpr.testFilter(current, root)
}

// ExistExpr represents an existence expression.
type ExistExpr struct {
	*Query
}

// testFilter returns true if e.Query selects any results from current or
// root.
func (e *ExistExpr) testFilter(current, root any) bool {
	return len(e.Select(current, root)) > 0
}

// writeTo writes a string representation of e to buf.
func (e *ExistExpr) writeTo(buf *strings.Builder) {
	buf.WriteString(e.Query.String())
}

// NotExistsExpr represents a nonexistence expression.
type NotExistsExpr struct {
	*Query
}

// writeTo writes a string representation of ne to buf.
func (ne NotExistsExpr) writeTo(buf *strings.Builder) {
	buf.WriteRune('!')
	buf.WriteString(ne.Query.String())
}

// testFilter returns true if ne.Query selects no results from current or
// root.
func (ne NotExistsExpr) testFilter(current, root any) bool {
	return len(ne.Select(current, root)) == 0
}
