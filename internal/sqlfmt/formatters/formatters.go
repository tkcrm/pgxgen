package formatters

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

// Options to define output format of Formatters
type Options struct {
	Padding    string        // Character sequence added as left padding on all lines, e.g. "" (none)
	Indent     string        // Character sequence used left indentation on indented clauses, e.g. "    " (4 spaces)
	Newline    string        // Character sequence used as line feeds, e.g. "\n" (newline character)
	Whitespace string        // Character sequence used as whitespace in SQL string, e.g. " " (single space)
	Dialect    lexer.Dialect // SQL dialect for formatting rules
}

// DefaultOptions returns a default options set for Formatters. Also used in unit tests.
func DefaultOptions() *Options {
	return &Options{
		Padding:    "",
		Indent:     "  ",
		Newline:    "\n",
		Whitespace: " ",
		Dialect:    lexer.DialectPostgreSQL,
	}
}

// Formatter interface. Example values of Formatter would be clause group or token
type Formatter interface {
	Format(buf *bytes.Buffer, parent []Formatter, parentIdx int) error
	AddIndent(lev int)
}

// Token wrapping lexer Token but extended with Options struct
type Token struct {
	lexer.Token
	*Options
}

// Format component accordingly with necessary indents, newlines,...
func (formatter Token) Format(buf *bytes.Buffer, parent []Formatter, parentIdx int) error {
	buf.WriteString(formatter.Value)
	return nil
}

// AddIndent increments indentation level by the given amount
func (formatter Token) AddIndent(lev int) {
}

// IsTieClauseStart determines if token type is included in TokenTypesOfTieClause
func (formatter Token) IsTieClauseStart() bool {
	return slices.Contains(lexer.TokenTypesOfTieClause, formatter.Type)
}

// IsLimitClauseStart determines token type is included in TokenTypesOfLimitClause
func (formatter Token) IsLimitClauseStart() bool {
	return slices.Contains(lexer.TokenTypesOfLimitClause, formatter.Type)
}

// IsJoinStart determines if token type is included in TokenTypesOfJoinMaker
func (formatter Token) IsJoinStart() bool {
	return slices.Contains(lexer.TokenTypesOfJoinMaker, formatter.Type)
}

// ContinueNewline returns true if a token should be moved to a new line
func (formatter Token) ContinueNewline() bool {
	ttypes := []lexer.TokenType{
		lexer.SELECT, lexer.FROM, lexer.ON, lexer.WHERE, lexer.HAVING, lexer.GROUP, lexer.ORDER, lexer.LIMIT, lexer.OFFSET,
		lexer.FETCH, lexer.RETURNING, lexer.USING, lexer.UNION, lexer.INTERSECT, lexer.EXCEPT, lexer.UNION,
		lexer.CREATE, lexer.UPDATE, lexer.SET, lexer.INSERT, lexer.VALUES, lexer.DELETE, lexer.DROP,
	}
	return slices.Contains(ttypes, formatter.Type)
}

// ContinueLine should be called on the last parent token to see, if a token should continue in the same line
func (formatter Token) ContinueLine() bool {
	return formatter.IsComparator() || formatter.Type == lexer.FROM || formatter.Type == lexer.WHERE ||
		formatter.Type == lexer.EXISTS || formatter.Type == lexer.AS || formatter.Type == lexer.IN ||
		formatter.Type == lexer.ON || formatter.Type == lexer.ANY || formatter.Type == lexer.ARRAY ||
		formatter.Type == lexer.AND || formatter.Type == lexer.OR
}

// IsComparator returns true if token is a comparator
func (formatter Token) IsComparator() bool {
	return formatter.Type == lexer.COMPARATOR
}

// process bracket, single quote and brace
// TODO: more elegant
func processPunctuation(rs []Formatter, WHITESPACE string) ([]Formatter, error) {
	var (
		result    []Formatter
		skipRange int
	)

	for i, v := range rs {
		if token, ok := v.(Token); ok {
			switch {
			case skipRange > 0:
				skipRange--
			case token.Type == lexer.STARTBRACE || token.Type == lexer.STARTBRACKET:
				surrounding, sr, err := extractSurroundingArea(rs[i:], WHITESPACE)
				if err != nil {
					return nil, err
				}
				result = append(result, Token{
					Options: token.Options,
					Token: lexer.Token{
						Type:  lexer.SURROUNDING,
						Value: surrounding,
					},
				})
				skipRange += sr
			default:
				result = append(result, token)
			}
		} else {
			result = append(result, v)
		}
	}
	return result, nil
}

// returns surrounding area including punctuation such as {xxx, xxx}
func extractSurroundingArea(rs []Formatter, WHITESPACE string) (string, int, error) {
	var (
		countOfStart int
		countOfEnd   int
		result       string
		skipRange    int
	)
	for i, r := range rs {
		if token, ok := r.(Token); ok {
			switch {
			case token.Type == lexer.COMMA || token.Type == lexer.STARTBRACKET || token.Type == lexer.STARTBRACE || token.Type == lexer.ENDBRACKET || token.Type == lexer.ENDBRACE:
				result += fmt.Sprint(token.Value)
				// for next token of StartToken
			case i == 1:
				result += fmt.Sprint(token.Value)
			default:
				result += fmt.Sprint(WHITESPACE + token.Value)
			}

			if token.Type == lexer.STARTBRACKET || token.Type == lexer.STARTBRACE || token.Type == lexer.STARTPARENTHESIS {
				countOfStart++
			}
			if token.Type == lexer.ENDBRACKET || token.Type == lexer.ENDBRACE || token.Type == lexer.ENDPARENTHESIS {
				countOfEnd++
			}
			if countOfStart == countOfEnd {
				break
			}
			skipRange++
		} else {
			// TODO: should support group type in surrounding area?
			// I have not encountered any groups in surrounding area so far
			return "", -1, errors.New("group type is not supposed be here")
		}
	}
	return result, skipRange, nil
}

func write(
	buf *bytes.Buffer,
	INDENT,
	NEWLINE,
	WHITESPACE string,
	token Token,
	previousToken,
	previousParentToken Token,
	indent int,
	hasMany bool,
) {
	switch {
	case token.ContinueNewline() && previousParentToken.Type != lexer.DELETE:
		fmt.Fprintf(buf, "%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value)
	case token.Type == lexer.DO:
		fmt.Fprintf(buf, "%s%s%s", NEWLINE, token.Value, WHITESPACE)
	case token.Type == lexer.WITH:
		fmt.Fprintf(buf, "%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value)

	// Write common token values
	case token.Type == lexer.COMMA: // Write comma token without whitespace
		buf.WriteString(token.Value)
	case strings.HasPrefix(token.Value, "::"):
		buf.WriteString(token.Value)
	default:

		// Move token to new line, because it cannot follow after single line comment
		if previousToken.Type == lexer.COMMENT && !strings.HasPrefix(previousToken.Value, "/*") {
			fmt.Fprintf(buf, "%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value)
			return
		}

		// Use newlines as separators
		if hasMany {
			if previousToken.ContinueNewline() {
				fmt.Fprintf(buf, "%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value)
			} else if previousToken.Type == lexer.AND || previousToken.Type == lexer.OR {
				fmt.Fprintf(buf, "%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value)
			} else {
				fmt.Fprintf(buf, "%s%s", WHITESPACE, token.Value)
			}
			return
		}

		fmt.Fprintf(buf, "%s%s", WHITESPACE, token.Value)
	}
}

func writeWithComma(
	buf *bytes.Buffer,
	INDENT,
	NEWLINE,
	WHITESPACE string,
	token,
	previousToken Token,
	indent int,
	position int,
	hasMany bool,
) {
	switch {
	case token.ContinueNewline():
		fmt.Fprintf(buf, "%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value)
	case position == 1 && hasMany && token.Type != lexer.COMMENT:
		fmt.Fprintf(buf, "%s%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), INDENT, token.Value)

	// Write comma token values or subsequent one
	case token.Type == lexer.COMMA: // Write comma token without whitespace
		buf.WriteString(token.Value)
	case previousToken.Type == lexer.COMMA && hasMany && token.Type != lexer.COMMENT:
		fmt.Fprintf(buf, "%s%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), INDENT, token.Value)

	// Write common token values
	case strings.HasPrefix(token.Value, "::"):
		buf.WriteString(token.Value)
	default:

		// Move token to new line, because it cannot follow after single line comment
		if previousToken.Type == lexer.COMMENT && !strings.HasPrefix(previousToken.Value, "/*") {
			fmt.Fprintf(buf, "%s%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), INDENT, token.Value)
			return
		}

		fmt.Fprintf(buf, "%s%s", WHITESPACE, token.Value)
	}
}
