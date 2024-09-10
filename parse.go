package jsonpath

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/theory/jsonpath/spec"
)

// ErrPathParse errors are returned for path parse errors.
var ErrPathParse = errors.New("jsonpath")

func makeError(tok token, msg string) error {
	return fmt.Errorf("%w: %v at position %v", ErrPathParse, msg, tok.pos+1)
}

// unexpected creates and returns an error for an unexpected token. For
// invalid tokens, the error will be as returned by the lexer. Otherwise, the
// error will "unexpected: $name".
func unexpected(tok token) error {
	if tok.tok == invalid {
		// Lex error message in the token value.
		return makeError(tok, tok.val)
	}
	return makeError(tok, "unexpected "+tok.name())
}

// Parse parses path, a JSON Path query string, into a Path. Returns a
// PathParseError on parse failure.
func Parse(path string) (*Path, error) {
	lex := newLexer(path)
	tok := lex.scan()

	switch tok.tok {
	case '$':
		// All path queries must start with $.
		q, err := parseQuery(true, lex)
		if err != nil {
			return nil, err
		}
		// Should have scanned to the end of input.
		if lex.r != eof {
			return nil, unexpected(lex.scan())
		}
		return New(q), nil
	case eof:
		// The token contained nothing.
		return nil, fmt.Errorf("%w: unexpected end of input", ErrPathParse)
	default:
		return nil, unexpected(tok)
	}
}

// parseQuery parses a query expression. lex.r should be set to $ (or,
// eventually, @) before calling. Returns the parsed Query.
func parseQuery(root bool, lex *lexer) (*spec.PathQuery, error) {
	segs := []*spec.Segment{}
	for {
		switch {
		case lex.r == '[':
			// Start of segment; scan selectors
			lex.scan()
			selectors, err := parseSelectors(lex)
			if err != nil {
				return nil, err
			}
			segs = append(segs, spec.Child(selectors...))
		case lex.r == '.':
			// Start of a name selector, wildcard, or descendant segment.
			lex.scan()
			if lex.r == '.' {
				// Consume `.` and parse descendant.
				lex.scan()
				seg, err := parseDescendant(lex)
				if err != nil {
					return nil, err
				}
				segs = append(segs, seg)
				continue
			}
			// Child segment with a name or wildcard selector.
			sel, err := parseNameOrWildcard(lex)
			if err != nil {
				return nil, err
			}
			segs = append(segs, spec.Child(sel))
		case lex.isBlankSpace(lex.r):
			switch lex.peekPastBlankSpace() {
			case '.', '[':
				lex.scanBlankSpace()
				continue
			}
			fallthrough
		default:
			// Done parsing.
			return spec.Query(root, segs), nil
		}
	}
}

// parseNameOrWildcard parses a name or '*' wildcard selector. Returns the
// parsed Selector.
//
//nolint:ireturn
func parseNameOrWildcard(lex *lexer) (spec.Selector, error) {
	switch tok := lex.scan(); tok.tok {
	case identifier:
		return spec.Name(tok.val), nil
	case '*':
		return spec.Wildcard, nil
	default:
		return nil, unexpected(tok)
	}
}

// parseDescendant parses a ".." descendant segment, which may be a bracketed
// segment or a wildcard or name selector segment. Returns the parsed Segment.
func parseDescendant(lex *lexer) (*spec.Segment, error) {
	switch tok := lex.scan(); tok.tok {
	case '[':
		// Start of segment; scan selectors
		selectors, err := parseSelectors(lex)
		if err != nil {
			return nil, err
		}
		return spec.Descendant(selectors...), nil
	case identifier:
		return spec.Descendant(spec.Name(tok.val)), nil
	case '*':
		return spec.Descendant(spec.Wildcard), nil
	default:
		return nil, unexpected(tok)
	}
}

// makeNumErr converts strconv.NumErrors to jsonpath errors.
func makeNumErr(tok token, err error) error {
	var numError *strconv.NumError
	if errors.As(err, &numError) {
		return makeError(tok, fmt.Sprintf(
			"cannot parse %q, %v",
			numError.Num, numError.Err.Error(),
		))
	}
	return makeError(tok, err.Error())
}

// parseSelectors parses Selectors from a bracket segment. lex.r should be '['
// before calling. Returns the Selectors parsed.
func parseSelectors(lex *lexer) ([]spec.Selector, error) {
	selectors := []spec.Selector{}
	for {
		switch tok := lex.scan(); tok.tok {
		case '?':
			filter, err := parseFilter(lex)
			if err != nil {
				return nil, err
			}
			selectors = append(selectors, filter)
		case '*':
			selectors = append(selectors, spec.Wildcard)
		case goString:
			selectors = append(selectors, spec.Name(tok.val))
		case integer:
			// Index or slice?
			if lex.skipBlankSpace() == ':' {
				// Slice.
				slice, err := parseSlice(lex, tok)
				if err != nil {
					return nil, err
				}
				selectors = append(selectors, slice)
			} else {
				// Index.
				idx, err := parsePathInt(tok)
				if err != nil {
					return nil, err
				}
				selectors = append(selectors, spec.Index(idx))
			}
		case ':':
			// Slice.
			slice, err := parseSlice(lex, tok)
			if err != nil {
				return nil, err
			}
			selectors = append(selectors, slice)
		case blankSpace:
			// Skip.
			continue
		default:
			return nil, unexpected(tok)
		}

		// Successfully parsed a selector. What's next?
		switch lex.skipBlankSpace() {
		case ',':
			// Consume the comma.
			lex.scan()
		case ']':
			// Consume and return.
			lex.scan()
			return selectors, nil
		default:
			// Anything else is an error.
			return nil, unexpected(lex.scan())
		}
	}
}

// parsePathInt parses an integer as used in index values and steps, which must be
// within the interval [-(253)+1, (253)-1].
func parsePathInt(tok token) (int64, error) {
	if tok.val == "-0" {
		return 0, makeError(tok, fmt.Sprintf(
			"invalid integer path value %q", tok.val,
		))
	}
	idx, err := strconv.ParseInt(tok.val, 10, 64)
	if err != nil {
		return 0, makeNumErr(tok, err)
	}
	const (
		minVal = -1<<53 + 1
		maxVal = 1<<53 - 1
	)
	if idx > maxVal || idx < minVal {
		return 0, makeError(tok, fmt.Sprintf(
			"cannot parse %q, value out of range",
			tok.val,
		))
	}
	return idx, nil
}

// parseSlice parses a slice selector, <start>:<end>:<step>. Returns the
// parsed SliceSelector.
func parseSlice(lex *lexer, tok token) (spec.SliceSelector, error) {
	var args [3]any

	// Parse the three parts: start, end, and step.
	i := 0
	for i < 3 {
		switch tok.tok {
		case ':':
			// Skip to the next index.
			i++
		case integer:
			// Parse the integer.
			num, err := parsePathInt(tok)
			if err != nil {
				return spec.SliceSelector{}, err
			}
			args[i] = int(num)
		default:
			// Nothing else allowed.
			return spec.SliceSelector{}, unexpected(tok)
		}

		// What's next?
		next := lex.skipBlankSpace()
		if next == ']' || next == ',' {
			// We've reached the end.
			return spec.Slice(args[0], args[1], args[2]), nil
		}
		tok = lex.scan()
	}

	// Never found the end of the slice.
	return spec.SliceSelector{}, unexpected(tok)
}

// parseFilter parses a [Filter] from Lex. A [Filter] consists of a single
// [LogicalOrExpr] (logical-or-expr).
func parseFilter(lex *lexer) (*spec.FilterSelector, error) {
	lor, err := parseLogicalOrExpr(lex)
	if err != nil {
		return nil, err
	}
	return spec.Filter(lor), nil
}

// parseLogicalOrExpr parses a [LogicalOrExpr] from lex. A [LogicalOrExpr] is
// made up of one or more [LogicalAndExpr] (logical-and-expr) separated by
// "||".
func parseLogicalOrExpr(lex *lexer) (spec.LogicalOr, error) {
	ands := []spec.LogicalAnd{}
	land, err := parseLogicalAndExpr(lex)
	if err != nil {
		return nil, err
	}

	ands = append(ands, land)
	lex.scanBlankSpace()
	for {
		if lex.r != '|' {
			break
		}
		lex.scan()
		next := lex.scan()
		if next.tok != '|' {
			return nil, makeError(next, fmt.Sprintf("expected '|' but found %v", next.name()))
		}
		land, err := parseLogicalAndExpr(lex)
		if err != nil {
			return nil, err
		}
		ands = append(ands, land)
	}

	return spec.LogicalOr(ands), nil
}

// parseLogicalAndExpr parses a [LogicalAndExpr] from lex. A [LogicalAndExpr]
// is made up of one or more [BasicExpr]s (basic-expr) separated by "&&".
func parseLogicalAndExpr(lex *lexer) (spec.LogicalAnd, error) {
	expr, err := parseBasicExpr(lex)
	if err != nil {
		return nil, err
	}

	ors := []spec.BasicExpr{expr}
	lex.scanBlankSpace()
	for {
		if lex.r != '&' {
			break
		}
		lex.scan()
		next := lex.scan()
		if next.tok != '&' {
			return nil, makeError(next, fmt.Sprintf("expected '&' but found %v", next.name()))
		}
		expr, err := parseBasicExpr(lex)
		if err != nil {
			return nil, err
		}
		ors = append(ors, expr)
	}

	return spec.LogicalAnd(ors), nil
}

// parseBasicExpr parses a [BasicExpr] from lex. A [BasicExpr] may be a
// parenthesized expression (paren-expr), comparison expression
// (comparison-expr), or test expression (test-expr).
//
//nolint:ireturn
func parseBasicExpr(lex *lexer) (spec.BasicExpr, error) {
	// Consume blank space.
	lex.skipBlankSpace()

	tok := lex.scan()
	switch tok.tok {
	case '!':
		if lex.skipBlankSpace() == '(' {
			// paren-expr
			lex.scan()
			return parseNotParenExpr(lex)
		}

		next := lex.scan()
		if next.tok == identifier {
			// test-expr or comparison-expr
			f, err := parseFunction(next, lex)
			if err != nil {
				return nil, err
			}
			return spec.NotFuncExpr{FunctionExpr: f}, nil
		}

		// test-expr or comparison-expr
		return parseNotExistsExpr(next, lex)
	case '(':
		return parseParenExpr(lex)
	case goString, integer, number, boolFalse, boolTrue, jsonNull:
		// comparison-expr
		left, err := parseLiteral(tok)
		if err != nil {
			return nil, err
		}
		return parseComparableExpr(left, lex)
	case identifier:
		if lex.r == '(' {
			return parseFunctionFilterExpr(tok, lex)
		}
	case '@', '$':
		q, err := parseFilterQuery(tok, lex)
		if err != nil {
			return nil, err
		}

		if sing := q.Singular(); sing != nil {
			switch lex.skipBlankSpace() {
			// comparison-expr
			case '=', '!', '<', '>':
				return parseComparableExpr(sing, lex)
			}
		}
		return &spec.ExistExpr{PathQuery: q}, nil
	}

	return nil, unexpected(tok)
}

// parseFunctionFilterExpr parses a [BasicExpr] (basic-expr) that starts with
// ident, which must be an identifier token that's expected to be the name of
// a function. The return value will be either a [FunctionExpr]
// (function-expr), if the function return value is a logical (boolean) value.
// Otherwise it will be a [ComparisonExpr] (comparison-expr), as long as the
// function call is compared to another expression. Any other configuration
// returns an error.
//
//nolint:ireturn
func parseFunctionFilterExpr(ident token, lex *lexer) (spec.BasicExpr, error) {
	f, err := parseFunction(ident, lex)
	if err != nil {
		return nil, err
	}

	if f.ResultType() == spec.FuncLogical {
		return f, nil
	}

	switch lex.skipBlankSpace() {
	case '=', '!', '<', '>':
		// comparison-expr
		return parseComparableExpr(f, lex)
	}

	return nil, makeError(lex.scan(), "missing comparison to function result")
}

// parseNotExistsExpr parses a [spec.NotExistsExpr] (non-existence) from lex.
func parseNotExistsExpr(tok token, lex *lexer) (*spec.NotExistsExpr, error) {
	q, err := parseFilterQuery(tok, lex)
	if err != nil {
		return nil, err
	}
	return spec.Nonexistence(q), nil
}

// parseFilterQuery parses a [*spec.Query] (rel-query / jsonpath-query) from
// lex.
func parseFilterQuery(tok token, lex *lexer) (*spec.PathQuery, error) {
	q, err := parseQuery(tok.tok == '$', lex)
	if err != nil {
		return nil, err
	}
	return q, nil
}

// parseLogicalOrExpr parses a [spec.LogicalOrExpr] from lex, which should
// return the next token after '(' from scan(). Returns an error if the
// expression does not end with a closing ')'.
func parseInnerParenExpr(lex *lexer) (spec.LogicalOr, error) {
	expr, err := parseLogicalOrExpr(lex)
	if err != nil {
		return nil, err
	}

	// Make sure we ended on a parenthesis.
	next := lex.scan()
	if next.tok != ')' {
		return nil, makeError(
			next, fmt.Sprintf("expected ')' but found %v", next.name()),
		)
	}

	return expr, nil
}

// parseParenExpr parses a [ParenExpr] (paren-expr) expression from lex, which
// should return the next token after '(' from scan(). Returns an error if the
// expression does not end with a closing ')'.
func parseParenExpr(lex *lexer) (*spec.ParenExpr, error) {
	expr, err := parseInnerParenExpr(lex)
	if err != nil {
		return nil, err
	}
	return spec.Paren(expr), nil
}

// parseParenExpr parses a [*spec.NotParenExpr] expression (logical-not-op
// paren-expression) from lex, which should return the next token after '('
// from scan(). Returns an error if the expression does not end with a closing
// ')'.
func parseNotParenExpr(lex *lexer) (*spec.NotParenExpr, error) {
	expr, err := parseInnerParenExpr(lex)
	if err != nil {
		return nil, err
	}
	return spec.NotParen(expr), nil
}

// parseFunction parses a function named tok.val from lex. tok should be the
// token just before the next call to lex.scan, and must be an identifier
// token naming the function. Returns an error if the function is not found in
// the registry or if arguments are invalid for the function.
func parseFunction(tok token, lex *lexer) (*spec.FunctionExpr, error) {
	paren := lex.scan() // Drop (
	args, err := parseFunctionArgs(lex)
	if err != nil {
		return nil, err
	}

	fe, err := spec.NewFunctionExpr(tok.val, args)
	if err != nil {
		// If the function is unknown, report error from function name.
		if errors.Is(err, spec.ErrUnregistered) {
			return nil, makeError(tok, err.Error())
		}
		// Otherwise report error from '('
		return nil, makeError(paren, err.Error())
	}
	return fe, nil
}

// parseFunctionArgs parses the comma-delimited arguments to a function from
// lex. Arguments may be one of literal, filter-query (including
// singular-query), logical-expr, or function-expr.
func parseFunctionArgs(lex *lexer) ([]spec.FunctionExprArg, error) {
	res := []spec.FunctionExprArg{}
	for {
		switch tok := lex.scan(); tok.tok {
		case goString, integer, number, boolFalse, boolTrue, jsonNull:
			// literal
			val, err := parseLiteral(tok)
			if err != nil {
				return nil, err
			}
			res = append(res, val)
		case '@', '$':
			// filter-query
			q, err := parseFilterQuery(tok, lex)
			if err != nil {
				return nil, err
			}

			res = append(res, q.Expression())

		case identifier:
			// function-expr

			if lex.skipBlankSpace() != '(' {
				return nil, unexpected(tok)
			}
			f, err := parseFunction(tok, lex)
			if err != nil {
				return nil, err
			}
			res = append(res, f)
		case blankSpace:
			// Skip.
			continue
		case ')':
			// All done.
			return res, nil
		case '!', '(':
			ors, err := parseLogicalOrExpr(lex)
			if err != nil {
				return nil, err
			}
			res = append(res, ors)
		}

		// Successfully parsed an argument. What's next?
		switch lex.skipBlankSpace() {
		case ',':
			// Consume the comma.
			lex.scan()
		case ')':
			// Consume and return.
			lex.scan()
			return res, nil
		default:
			// Anything else is an error.
			return nil, unexpected(lex.scan())
		}
	}
}

// parseLiteral parses the literal value from tok into native Go values and
// returns them as spec.LiteralArg. tok.tok must be one of goString, integer,
// number, boolFalse, boolTrue, or jsonNull.
func parseLiteral(tok token) (*spec.LiteralArg, error) {
	switch tok.tok {
	case goString:
		return spec.Literal(tok.val), nil
	case integer:
		integer, err := strconv.ParseInt(tok.val, 10, 64)
		if err != nil {
			return nil, makeNumErr(tok, err)
		}
		return spec.Literal(integer), nil
	case number:
		num, err := strconv.ParseFloat(tok.val, 64)
		if err != nil {
			return nil, makeNumErr(tok, err)
		}
		return spec.Literal(num), nil
	case boolTrue:
		return spec.Literal(true), nil
	case boolFalse:
		return spec.Literal(false), nil
	case jsonNull:
		return spec.Literal(nil), nil
	default:
		return nil, unexpected(tok)
	}
}

// parseComparableExpr parses a [ComparisonExpr] (comparison-expr) from lex.
func parseComparableExpr(left spec.CompVal, lex *lexer) (*spec.ComparisonExpr, error) {
	// Skip blank space.
	lex.skipBlankSpace()

	op, err := parseCompOp(lex)
	if err != nil {
		return nil, err
	}

	// Skip blank space.
	lex.skipBlankSpace()

	right, err := parseComparableVal(lex.scan(), lex)
	if err != nil {
		return nil, err
	}

	return &spec.ComparisonExpr{Left: left, Op: op, Right: right}, nil
}

// parseComparableVal parses a [CompVal] (comparable) from lex.
//
//nolint:ireturn
func parseComparableVal(tok token, lex *lexer) (spec.CompVal, error) {
	switch tok.tok {
	case goString, integer, number, boolFalse, boolTrue, jsonNull:
		// literal
		return parseLiteral(tok)
	case '@', '$':
		// singular-query
		return parseSingularQuery(tok, lex)
	case identifier:
		// function-expr
		if lex.r != '(' {
			return nil, unexpected(tok)
		}
		f, err := parseFunction(tok, lex)
		if err != nil {
			return nil, err
		}
		if f.ResultType() == spec.FuncLogical {
			return nil, makeError(tok, "cannot compare result of logical function")
		}
		return f, nil
	default:
		return nil, unexpected(tok)
	}
}

// parseCompOp pares a [CompOp] (comparison-op) from lex.
func parseCompOp(lex *lexer) (spec.CompOp, error) {
	tok := lex.scan()
	switch tok.tok {
	case '=':
		if lex.r == '=' {
			lex.scan()
			return spec.EqualTo, nil
		}
	case '!':
		if lex.r == '=' {
			lex.scan()
			return spec.NotEqualTo, nil
		}
	case '<':
		if lex.r == '=' {
			lex.scan()
			return spec.LessThanEqualTo, nil
		}
		return spec.LessThan, nil
	case '>':
		if lex.r == '=' {
			lex.scan()
			return spec.GreaterThanEqualTo, nil
		}
		return spec.GreaterThan, nil
	}

	return 0, makeError(tok, "invalid comparison operator")
}

// parseSingularQuery parses a [spec.SingularQueryExpr] (singular-query) from
// lex. A singular query consists only of single-selector nodes.
func parseSingularQuery(startToken token, lex *lexer) (*spec.SingularQueryExpr, error) {
	selectors := []spec.Selector{}
	for {
		switch lex.r {
		case '[':
			// Index or name selector.
			lex.skipBlankSpace()
			lex.scan()
			switch tok := lex.scan(); tok.tok {
			case goString:
				selectors = append(selectors, spec.Name(tok.val))
			case integer:
				idx, err := parsePathInt(tok)
				if err != nil {
					return nil, err
				}
				selectors = append(selectors, spec.Index(idx))
			default:
				return nil, unexpected(tok)
			}
			// Look for closing bracket.
			lex.skipBlankSpace()
			tok := lex.scan()
			if tok.tok != ']' {
				return nil, unexpected(tok)
			}
		case '.':
			// Start of a name selector.
			lex.scan()
			tok := lex.scan()
			if tok.tok != identifier {
				return nil, unexpected(tok)
			}
			selectors = append(selectors, spec.Name(tok.val))
		default:
			// Done parsing.
			return spec.SingularQuery(startToken.tok == '$', selectors), nil
		}
	}
}
