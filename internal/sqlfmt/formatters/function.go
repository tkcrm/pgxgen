package formatters

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

// Function group formatter
type Function struct {
	Elements     []Formatter
	IndentLevel  int
	*Options     // Options used later to format element
	IsColumnArea bool
}

// Format component accordingly with necessary indents, newlines,...
func (formatter *Function) Format(buf *bytes.Buffer, parent []Formatter, parentIdx int) error {

	// Prepare short variables for better visibility
	var WHITESPACE = formatter.Whitespace

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
			formatter.writeFunction(buf, token, previousToken, formatter.IndentLevel, formatter.IsColumnArea)
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
func (formatter *Function) AddIndent(lev int) {
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

func (formatter *Function) writeFunction(buf *bytes.Buffer, token, previousToken Token, indent int, isColumnArea bool) {

	// Prepare short variables for better visibility
	var INDENT = formatter.Indent
	var NEWLINE = formatter.Newline
	var WHITESPACE = formatter.Whitespace

	// Write element
	switch {
	case token.Type == lexer.FUNCTION && isColumnArea: // Write function name token to new line in SELECT clause
		buf.WriteString(fmt.Sprintf("%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value))
	case token.Type == lexer.STARTPARENTHESIS || token.Type == lexer.ENDPARENTHESIS: // Write function's parentheses without whitespace
		buf.WriteString(fmt.Sprintf("%s", token.Value))
	case previousToken.Type == lexer.STARTPARENTHESIS: // Write first function value token without whitespace
		buf.WriteString(fmt.Sprintf("%s", token.Value))

	// Write common token values
	case token.Type == lexer.COMMA: // Write comma token without whitespace
		buf.WriteString(fmt.Sprintf("%s", token.Value))
	case strings.HasPrefix(token.Value, "::"): // Write cast token without whitespace
		buf.WriteString(fmt.Sprintf("%s", token.Value))
	default:

		// Move token to new line, because it cannot follow after single line comment
		if previousToken.Type == lexer.COMMENT && !strings.HasPrefix(previousToken.Value, "/*") {
			buf.WriteString(fmt.Sprintf("%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value))
			return
		}

		buf.WriteString(fmt.Sprintf("%s%s", WHITESPACE, token.Value))
	}
}
