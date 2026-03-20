package formatters

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

// Parenthesis group formatter
type Parenthesis struct {
	Elements         []Formatter
	IndentLevel      int
	*Options         // Options used later to format element
	IsColumnArea     bool
	PositionInParent int
}

// Format component accordingly with necessary indents, newlines,...
func (formatter *Parenthesis) Format(buf *bytes.Buffer, parent []Formatter, parentIdx int) error {

	// Prepare short variables for better visibility
	var INDENT = formatter.Indent
	var NEWLINE = formatter.Newline
	var WHITESPACE = formatter.Whitespace

	// Preprocess punctuation and enrich with surrounding information
	elements, err := processPunctuation(formatter.Elements, WHITESPACE)
	if err != nil {
		return err
	}

	// Get last token written by parent
	var previousParentToken Token
	if len(parent) > parentIdx && parentIdx > 0 {
		if token, ok := parent[parentIdx-1].(Token); ok {
			previousParentToken = token
		}
	}

	// Decide whether to start parenthesis in same line or next one
	startSameLine := true // By default, start in same line
	if previousParentToken.ContinueLine() {
		startSameLine = true
	} else if previousParentToken.Type == lexer.STARTPARENTHESIS { // Nested second parenthesis should be moved to a new line and indented
		startSameLine = false
	} else if previousParentToken.Type == lexer.COMMENT { // Preceding comment should move parenthesis to new line and indent
		startSameLine = false
	} else if formatter.IsColumnArea && formatter.PositionInParent == 0 { // Parenthesis in column area in first column might be some special select clause
		startSameLine = false
	}

	// Check if parenthesis group has nested element
	var endSameLine = true
	for _, el := range elements {
		switch el.(type) {
		case Token:
		case Formatter:
			endSameLine = false
		}
	}

	// Check if there are type definitions in the list of values
	// This is a special format case for CREATE TABLE queries
	var hasTypeDefinitions = false
	for _, el := range elements {
		switch t := el.(type) {
		case Token:
			if t.Type == lexer.TYPE {
				hasTypeDefinitions = true
			}
		}
	}

	// Iterate and write elements to the buffer. Recursively step into nested elements.
	var previousToken Token
	for i, el := range elements {

		// Write element or recursively call its Format function
		if token, ok := el.(Token); ok {
			writeParenthesis(buf, INDENT, NEWLINE, WHITESPACE, token, previousToken, formatter.IndentLevel, i, startSameLine, endSameLine, hasTypeDefinitions)
		} else {

			// Increment indent, as everything within PARENTHESIS should be indented
			el.AddIndent(1)

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
func (formatter *Parenthesis) AddIndent(lev int) {
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

func writeParenthesis(
	buf *bytes.Buffer,
	INDENT,
	NEWLINE,
	WHITESPACE string,
	token,
	previousToken Token,
	indent int,
	position int,
	startSameLine,
	endSameLine bool,
	containsTypeDefinitions bool,
) {

	// Write element
	if startSameLine && endSameLine { // Parenthesis starts and ends in the same line - one-liner case
		switch {
		case token.Type == lexer.STARTPARENTHESIS:
			buf.WriteString(fmt.Sprintf("%s%s", WHITESPACE, token.Value))
			return
		case position == 1 && !containsTypeDefinitions:
			buf.WriteString(fmt.Sprintf("%s", token.Value))
			return
		case token.Type == lexer.ENDPARENTHESIS:
			buf.WriteString(fmt.Sprintf("%s", token.Value))
			return
		}
	} else if startSameLine && !endSameLine { // Parenthesis starts in the same line but ends in new line
		switch {
		case token.Type == lexer.STARTPARENTHESIS:
			buf.WriteString(fmt.Sprintf("%s%s", WHITESPACE, token.Value))
			return
		case position == 1 && token.Type != lexer.COMMENT:
			buf.WriteString(fmt.Sprintf("%s%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), INDENT, token.Value))
			return
		case token.Type == lexer.ENDPARENTHESIS:
			buf.WriteString(fmt.Sprintf("%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value))
			return
		}
	} else if !startSameLine && !endSameLine { // Parenthesis starts and ends in new lines
		switch {
		case token.Type == lexer.STARTPARENTHESIS:
			buf.WriteString(fmt.Sprintf("%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value))
			return
		case position == 1 && token.Type != lexer.COMMENT:
			buf.WriteString(fmt.Sprintf("%s%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), INDENT, token.Value))
			return
		case token.Type == lexer.ENDPARENTHESIS:
			buf.WriteString(fmt.Sprintf("%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), token.Value))
			return
		}
	} else { // `if !startSameLine && endSameLine`
		switch {
		case token.Type == lexer.STARTPARENTHESIS:
			buf.WriteString(fmt.Sprintf("%s%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), INDENT, token.Value))
			return
		case position == 1 && !containsTypeDefinitions:
			buf.WriteString(fmt.Sprintf("%s", token.Value))
			return
		case token.Type == lexer.ENDPARENTHESIS:
			buf.WriteString(fmt.Sprintf("%s", token.Value))
			return
		}
	}

	// Write common token values
	switch {

	// Write comma token values or subsequent one
	case token.Type == lexer.COMMA: // Write comma token without whitespace
		buf.WriteString(fmt.Sprintf("%s", token.Value))
	case previousToken.Type == lexer.COMMA && containsTypeDefinitions:
		buf.WriteString(fmt.Sprintf("%s%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), INDENT, token.Value))

	// Write common token values
	case strings.HasPrefix(token.Value, "::"):
		buf.WriteString(fmt.Sprintf("%s", token.Value))
	default:

		// Move token to new line, because it cannot follow after single line comment
		if previousToken.Type == lexer.COMMENT && !strings.HasPrefix(previousToken.Value, "/*") {
			buf.WriteString(fmt.Sprintf("%s%s%s%s", NEWLINE, strings.Repeat(INDENT, indent), INDENT, token.Value))
			return
		}

		buf.WriteString(fmt.Sprintf("%s%s", WHITESPACE, token.Value))
	}
}
