package parser

// EOF is defined to get a char
const EOF rune = 0

// MinusSign
const MinusSign = "--"

// LeftBracket
const LeftBracket string = "[["

// RightBracket
const RightBracket string = "]]"

// NEWLINE
const NEWLINE string = "\n"

// TokenType defines a token
type TokenType int

const (
	// TError defines a parse error
	TError TokenType = iota

	// TEOF is End of File
	TEOF

	// TSLComment is --
	TSLComment

	// TRMLComment is --[[
	TRMLComment

	// TLMLComment is --]]
	TLMLComment

	// TNewline is \n or \r
	TNewline

	// TComment is the comment
	TComment
)

// Token is a parsed token
type Token struct {
	Type  TokenType
	Value string
	Row   int
}
