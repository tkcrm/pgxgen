package formatters

import (
	"bytes"
	"fmt"
	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
	"strings"
)

// And formatter
type And struct {
	Elements    []Formatter
	IndentLevel int
	*Options    // Options used later to format element
	SameLine    bool
}

// Format component accordingly with necessary indents, newlines,...
func (formatter *And) Format(buf *bytes.Buffer, parent []Formatter, parentIdx int) error {

	// Prepare short variables for better visibility
	var INDENT = formatter.Indent
	var NEWLINE = formatter.Newline
	var WHITESPACE = formatter.Whitespace

	// Preprocess punctuation and enrich with surrounding information
	elements, err := processPunctuation(formatter.Elements, WHITESPACE)
	if err != nil {
		return err
	}

	// Check if parent's first token is indicating Join
	var isPartOfJoin = false
	if parent != nil {
		if t, ok := parent[0].(Token); ok {
			if t.IsJoinStart() {
				isPartOfJoin = true
			}
		}
	}

	// Iterate and write elements to the buffer. Recursively step into nested elements.
	var previousToken Token
	for i, el := range elements {

		// Write element or recursively call its Format function
		if token, ok := el.(Token); ok {
			writeAnd(buf, INDENT, NEWLINE, WHITESPACE, token, previousToken, formatter.IndentLevel, formatter.SameLine, isPartOfJoin)
		} else {

			// Recursively format nested elements
			_ = el.Format(buf, elements, i)
		}

		// Remember last Token element
		if token, ok := el.(Token); ok {
			previousToken = token
		} else {
			previousToken = Token{}
		}
	}

	// Return nil and continue with parent Formatter
	return nil
}

// AddIndent increments indentation level by the given amount
func (formatter *And) AddIndent(lev int) {
	formatter.IndentLevel += lev

	// Preprocess punctuation and enrich with surrounding information
	elements, err := processPunctuation(formatter.Elements, formatter.Whitespace)
	if err != nil {
		elements = formatter.Elements
	}

	// Iterate and increase indent of child elements too
	for _, el := range elements {
		el.AddIndent(lev)
	}
}

func writeAnd(
	buf *bytes.Buffer,
	INDENT,
	NEWLINE,
	WHITESPACE string,
	token,
	previousToken Token,
	indent int,
	sameLine bool,
	isPartOfJoin bool,
) {

	// Print to same line with WHITESPACE
	switch {
	case strings.HasPrefix(token.Value, "::"): // Write cast token without whitespace
		buf.WriteString(fmt.Sprintf("%s", token.Value))
	case sameLine || isPartOfJoin:
		buf.WriteString(fmt.Sprintf("%s%s", WHITESPACE, token.Value))
	case token.Type == lexer.AND || token.Type == lexer.OR: // Start of where clause
		buf.WriteString(fmt.Sprintf("%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value))

	// Write common token values
	default:

		// Move token to new line, because it cannot follow after single line comment
		if previousToken.Type == lexer.COMMENT && !strings.HasPrefix(previousToken.Value, "/*") {
			buf.WriteString(fmt.Sprintf("%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value))
			return
		}

		buf.WriteString(fmt.Sprintf("%s%s", WHITESPACE, token.Value))
	}
}
