package formatters

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

// Insert group formatter
type Insert struct {
	Elements    []Formatter
	IndentLevel int
	*Options    // Options used later to format element
}

// Format component accordingly with necessary indents, newlines,...
func (formatter *Insert) Format(buf *bytes.Buffer, parent []Formatter, parentIdx int) error {
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
			formatter.WriteInsert(buf, token, previousToken, formatter.IndentLevel)
		} else {

			// Break parenthesis group (list of values) into new line
			switch v := el.(type) {
			case *Parenthesis:
				v.IsColumnArea = true
			}

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
func (formatter *Insert) AddIndent(lev int) {
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

func (formatter *Insert) WriteInsert(buf *bytes.Buffer, token, previousToken Token, indent int) {
	// Prepare short variables for better visibility
	INDENT := formatter.Indent
	NEWLINE := formatter.Newline
	WHITESPACE := formatter.Whitespace

	switch {
	case token.ContinueNewline():
		fmt.Fprintf(buf, "%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value)

	// Write common token values
	default:

		// Move token to new line, because it cannot follow after single line comment
		if previousToken.Type == lexer.COMMENT && !strings.HasPrefix(previousToken.Value, "/*") {
			fmt.Fprintf(buf, "%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value)
			return
		}

		fmt.Fprintf(buf, "%s%s", WHITESPACE, token.Value)
	}
}
