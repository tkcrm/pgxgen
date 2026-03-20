package formatters

import (
	"bytes"
)

// Or formatter
type Or struct {
	Elements    []Formatter
	IndentLevel int
	*Options    // Options used later to format element
	SameLine    bool
}

// Format component accordingly with necessary indents, newlines,...
func (formatter *Or) Format(buf *bytes.Buffer, parent []Formatter, parentIdx int) error {

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
			writeAnd(buf, INDENT, NEWLINE, WHITESPACE, token, previousToken, formatter.IndentLevel, formatter.SameLine, isPartOfJoin) // OR is not different to an AND in regard to formatting
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
func (formatter *Or) AddIndent(lev int) {
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
