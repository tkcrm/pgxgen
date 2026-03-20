package formatters

import (
	"bytes"
	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

// Subquery formatters
type Subquery struct {
	Elements     []Formatter
	IndentLevel  int
	*Options     // Options used later to format element
	IsColumnArea bool
}

// Format component accordingly with necessary indents, newlines,...
func (formatter *Subquery) Format(buf *bytes.Buffer, parent []Formatter, parentIdx int) error {

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
	} else if formatter.IsColumnArea { // Subqueries in SELECT columns should be moved to a new line
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

	// Iterate and write elements to the buffer. Recursively step into nested elements.
	var previousToken Token
	for i, el := range elements {

		// Write element or recursively call its Format function
		if token, ok := el.(Token); ok {
			writeParenthesis(buf, INDENT, NEWLINE, WHITESPACE, token, previousToken, formatter.IndentLevel, i, startSameLine, endSameLine, false) // Subquery is not different to a parenthesis group in regard to formatting
		} else {

			// Increment indent, as everything within SUBQUERY (similar to parenthesis) should be indented
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
func (formatter *Subquery) AddIndent(lev int) {
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
