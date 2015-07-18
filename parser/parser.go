package parser

import (
	"fmt"
	"io/ioutil"
)

// Comment contains information about a comment
type Comment struct {
	Value string
	Row   int
}

// ErrNoMatch is returned when no matching comment can be found for a row
var ErrNoMatch = fmt.Errorf("No matching comment")

// Comments that has been parsed
type Comments []Comment

// Get returns a Comment or an error
func (c Comments) Get(row int) (*Comment, error) {
	result := &Comment{Row: row}
	for i := len(c) - 1; i >= 0; i-- {
		if c[i].Row != row {
			continue
		}
		result.Value = result.Value + c[i].Value
		row--
	}

	if result.Value == "" {
		return nil, ErrNoMatch
	}

	return result, nil
}

// String parses a string
func String(str string) Comments {
	var token Token

	comments := make(Comments, 0)

	l := BeginLexing("", str)

	for {
		token = l.NextToken()

		if token.Type == TComment {
			comment := Comment{Value: token.Value, Row: token.Row}
			comments = append(comments, comment)
		}

		if token.Type == TEOF {
			return comments
		}

	}
}

// File parses a file
func File(file string) (Comments, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse file: %v", err)
	}

	return String(string(buf)), nil
}
