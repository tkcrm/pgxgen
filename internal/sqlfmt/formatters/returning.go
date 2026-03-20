package formatters

import (
	"bytes"
	"github.com/tkcrm/pgxgen/internal/sqlfmt/lexer"
)

const maxReturningClausesPerLine = 2

// Returning group formatter
type Returning struct {
	Elements    []Formatter
	IndentLevel int
	*Options    // Options used later to format element
}

// Format component accordingly with necessary indents, newlines,...
func (formatter *Returning) Format(buf *bytes.Buffer, parent []Formatter, parentIdx int) error {

	// Prepare short variables for better visibility
	var INDENT = formatter.Indent
	var NEWLINE = formatter.Newline
	var WHITESPACE = formatter.Whitespace

	// Preprocess punctuation and enrich with surrounding information
	elements, err := processPunctuation(formatter.Elements, WHITESPACE)
	if err != nil {
		return err
	}

	// Check how many clauses there are. Linebreak if too many
	var clauses = 0
	for _, el := range elements {
		switch t := el.(type) {
		case Token:
			if t.Type == lexer.IDENT {
				clauses++
			} else if t.Type == lexer.COMMENT {
				clauses = 999 // Format like if there were many clauses to make space for comments
			}
		}
	}

	// Iterate and write elements to the buffer. Recursively step into nested elements.
	var hasMany = clauses > maxReturningClausesPerLine
	var previousToken Token
	for i, el := range elements {

		// Write element or recursively call its Format function
		if token, ok := el.(Token); ok {
			writeWithComma(buf, INDENT, NEWLINE, WHITESPACE, token, previousToken, formatter.IndentLevel, i, hasMany)
		} else {

			// Increment indent, if ORDER clauses should be written into new lines
			if hasMany {
				el.AddIndent(1)
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
func (formatter *Returning) AddIndent(lev int) {
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
