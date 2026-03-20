package formatters

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

// Join group formatter
type Join struct {
	Elements    []Formatter
	IndentLevel int
	*Options    // Options used later to format element
}

// Format component accordingly with necessary indents, newlines,...
func (formatter *Join) Format(buf *bytes.Buffer, parent []Formatter, parentIdx int) error {
	// Prepare short variables for better visibility
	WHITESPACE := formatter.Whitespace

	// Preprocess punctuation and enrich with surrounding information
	elements, err := processPunctuation(formatter.Elements, WHITESPACE)
	if err != nil {
		return err
	}

	// Iterate and write elements to the buffer. Recursively step into nested elements.
	var previousToken Token
	for i, el := range elements {

		// Write element or recursively call its Format function
		if token, ok := el.(Token); ok {
			formatter.writeJoin(buf, token, previousToken, formatter.IndentLevel, i)
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
func (formatter *Join) AddIndent(lev int) {
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

func (formatter *Join) writeJoin(buf *bytes.Buffer, token, previousToken Token, indent int, position int) {
	// Prepare short variables for better visibility
	INDENT := formatter.Indent
	NEWLINE := formatter.Newline
	WHITESPACE := formatter.Whitespace

	// Write element
	switch {
	case position == 0 && token.IsJoinStart():
		fmt.Fprintf(buf, "%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value)
	case token.Type == lexer.ON || token.Type == lexer.USING:
		fmt.Fprintf(buf, " %s", token.Value)

	// Write common token values
	case strings.HasPrefix(token.Value, "::"):
		buf.WriteString(token.Value)
	default:

		// Move token to new line, because it cannot follow after single line comment
		if previousToken.Type == lexer.COMMENT && !strings.HasPrefix(previousToken.Value, "/*") {
			fmt.Fprintf(buf, "%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value)
			return
		}

		fmt.Fprintf(buf, "%s%s", WHITESPACE, token.Value)
	}
}
