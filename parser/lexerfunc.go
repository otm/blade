package parser

import "strings"

// LexBegin is the initial lexer function
func LexBegin(lexer *Lexer) LexFn {
	lexer.SkipWhitespace()

	if strings.HasPrefix(lexer.InputToEnd(), MinusSign) {
		return LexSingleLineComment
	}
	return LexCode
}

// LexSingleLineComment lexer
func LexSingleLineComment(lexer *Lexer) LexFn {
	lexer.Next()
	lexer.Next()
	lexer.Ignore()
	if strings.HasPrefix(lexer.InputToEnd(), LeftBracket) {
		return LexMultiLineComment(lexer)
	}

	for {
		if strings.HasPrefix(lexer.InputToEnd(), NEWLINE) {
			//lexer.Backup()
			lexer.Emit(TComment)
			//lexer.Next()
			return LexBegin(lexer)
		}

		lexer.Next()

		if lexer.IsEOF() {
			lexer.Emit(TComment)
			// TODO: Lexer Stop
			lexer.Emit(TEOF)
			lexer.Shutdown()
			return nil
		}
	}
}

// LexMultiLineComment lexer
func LexMultiLineComment(lexer *Lexer) LexFn {
	lexer.Pos += len(LeftBracket)
	lexer.Ignore()

	for {
		if strings.HasPrefix(lexer.InputToEnd(), MinusSign+RightBracket) {
			lexer.Emit(TComment)
			return LexBegin(lexer)
		}

		lexer.Inc()

		if lexer.IsEOF() {
			panic("EndOfFile")
		}
	}
}

// LexCode parsers random code
func LexCode(lexer *Lexer) LexFn {
	for {
		if strings.HasPrefix(lexer.InputToEnd(), MinusSign) {
			lexer.Start = lexer.Pos
			return LexSingleLineComment(lexer)
		}

		if strings.HasPrefix(lexer.InputToEnd(), NEWLINE) {
			lexer.Next()
			lexer.Start = lexer.Pos
			return LexCode(lexer)
		}

		lexer.Next()

		if lexer.IsEOF() {
			lexer.Emit(TEOF)
			lexer.Shutdown()
			return nil
		}
	}
}
