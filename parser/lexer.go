package parser

import (
	"unicode"
	"unicode/utf8"
)

// BeginLexing returns a new lexer
func BeginLexing(name, input string) *Lexer {
	l := &Lexer{
		Name:   name,
		Input:  input,
		State:  LexBegin,
		Tokens: make(chan Token, 3),
		Row:    1,
	}

	return l
}

// LexFn defines a lexer
type LexFn func(*Lexer) LexFn

// Lexer defines our parser
type Lexer struct {
	Name   string
	Input  string
	Tokens chan Token
	State  LexFn

	Start int
	Pos   int
	Width int
	Row   int
}

// Emit puts a token onto the token channel. The value of l token is
// read from the input based on the current lexer position.
func (l *Lexer) Emit(tokenType TokenType) {
	l.Tokens <- Token{Type: tokenType, Value: l.Input[l.Start:l.Pos], Row: l.Row}
	l.Start = l.Pos
}

// Inc increments the position
func (l *Lexer) Inc() {
	l.Pos++
	if l.Pos >= utf8.RuneCountInString(l.Input) {
		l.Emit(TEOF)
	}

	result, _ := utf8.DecodeRuneInString(l.Input[l.Pos:])

	if result == '\n' {
		l.Row++
	}
}

// InputToEnd returns a slice of the input from the current lexer position
// to the end of the input string.
func (l *Lexer) InputToEnd() string {
	return l.Input[l.Pos:]
}

// Dec decrements the possition
func (l *Lexer) Dec() {
	l.Pos--

	result, _ := utf8.DecodeRuneInString(l.Input[l.Pos:])
	if result == '\n' {
		l.Row--
	}
}

// Ignore ignors the currently parsed data
func (l *Lexer) Ignore() {
	l.Start = l.Pos
}

// Peek returns next rune without advancing the parser
func (l *Lexer) Peek() rune {
	rune := l.Next()
	l.Backup()
	return rune
}

// Backup moves parser to last read token
func (l *Lexer) Backup() {
	l.Pos -= l.Width

	result, _ := utf8.DecodeRuneInString(l.Input[l.Pos:])

	if result == '\n' {
		l.Row--
	}

}

// Next reads the next rune (character) from the input stream
// and advances the lexer position.
func (l *Lexer) Next() rune {
	if l.Pos >= utf8.RuneCountInString(l.Input) {
		l.Width = 0
		return EOF
	}

	result, width := utf8.DecodeRuneInString(l.Input[l.Pos:])

	if result == '\n' {
		l.Row++
	}

	l.Width = width
	l.Pos += l.Width
	return result
}

// IsEOF checks if we reached the end of file
func (l *Lexer) IsEOF() bool {
	return l.Pos >= len(l.Input)
}

// Shutdown the parser
func (l *Lexer) Shutdown() {
	close(l.Tokens)
}

// Run the parser
func (l *Lexer) Run() {
	for state := LexBegin; state != nil; {
		state = state(l)
	}

	l.Shutdown()
}

// NextToken returns the next token from the channel
func (l *Lexer) NextToken() Token {
	for {
		select {
		case token := <-l.Tokens:
			return token
		default:
			l.State = l.State(l)
		}
	}
}

// SkipWhitespace skips whitespaces
func (l *Lexer) SkipWhitespace() {
	for {
		ch := l.Next()

		if !(unicode.IsSpace(ch) || ch == '\n') {
			l.Dec()
			l.Ignore()
			return
		}

		if ch == EOF {
			l.Emit(TEOF)
			return
		}
	}
}
