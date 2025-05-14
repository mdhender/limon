// limon - a parser generator
//
// Copyright (c) 2019 Michael D Henderson
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// Some source from https://golang.org/src/text/scanner/scanner.go

package scanner

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"unicode"
	"unicode/utf8"
)

// Token is returned from the scanner.
type Token struct {
	ID             int
	Line, Col, Pos int
	Value          []byte
}

// Tokenizer is the scanner. It returns a stream of tokens or an error.
type Tokenizer struct {
	buf       []byte
	pos       int
	Line, Col int
}

// enums for token ID
const (
	COMMENT int = iota + 1
	LEFT_CURLY_BRACE
	RIGHT_CURLY_BRACE
	MULTILINE_COMMENT
	NEWLINE
	SPECIAL_DIRECTIVE_DEFAULT_TYPE
	SPECIAL_DIRECTIVE_EXTRA_CONTEXT
	SPECIAL_DIRECTIVE_SYNTAX_ERROR
	SPECIAL_DIRECTIVE_TOKEN_PREFIX
	SPECIAL_DIRECTIVE_TOKEN_TYPE
	TEXT
	UNKNOWN
	WHITESPACE
)

// Create returns an initialized scanner.
func Create(filename string) (*Tokenizer, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	if err = testBuffer(buf); err != nil {
		return nil, err
	}
	t := Tokenizer{
		buf:  buf,
		Line: 1,
		Col:  1,
	}
	return &t, nil
}

// Free releases all memory used by the scanner.
func (t *Tokenizer) Free() {
}

// GetNextToken returns the next token in the input.
func (t *Tokenizer) GetNextToken() (*Token, error) {
	tok := Token{ID: UNKNOWN, Pos: t.pos, Line: t.Line, Col: t.Col}

	if !(t.pos < len(t.buf)) {
		tok.ID = EOF
		return &tok, nil
	}

	var r rune
	var w int
	state := "start"
	for {
		if t.pos < len(t.buf) {
			r, w = utf8.DecodeRune(t.buf[t.pos:])
		} else {
			r, w = utf8.RuneError, 0
		}

		switch r {
		case utf8.RuneError:
			switch state {
			case "inSingleLineComment":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readingText":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readingWhiteSpace":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			default:
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, fmt.Errorf("%d:%d: unexpected end-of-input (%s)", t.Line, t.Col, state)
			}
		case '\n':
			switch state {
			case "start":
				tok.ID = NEWLINE
				t.pos, t.Line, t.Col = t.pos+w, t.Line+1, 1
				return &tok, nil
			case "inMultiLineComment":
				t.pos, t.Line, t.Col = t.pos+w, t.Line+1, 1
			case "inSingleLineComment":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readingText":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readingWhiteSpace":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readSplatInMultiLineComment":
				t.pos, t.Line, t.Col = t.pos+w, t.Line+1, 1
				state = "inMultiLineComment"
			default:
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, fmt.Errorf("%d:%d: unexpected end-of-line (%s)", t.Line, t.Col, state)
			}
		case '/':
			switch state {
			case "inSingleLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
			case "readingText":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readingWhiteSpace":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readSlash":
				tok.ID = COMMENT
				t.pos, t.Col = t.pos+w, t.Col+1
				state = "inSingleLineComment"
			case "readSplatInMultiLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "start":
				state = "readSlash"
				t.pos, t.Col = t.pos+w, t.Col+1
			default:
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, fmt.Errorf("%d:%d: unexpected character %q (%s)", t.Line, t.Col, string(r), state)
			}
		case '*':
			switch state {
			case "inMultiLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
				state = "readSplatInMultiLineComment"
			case "inSingleLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
			case "readingText":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readingWhiteSpace":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readSlash":
				tok.ID = MULTILINE_COMMENT
				t.pos, t.Col = t.pos+w, t.Col+1
				state = "inMultiLineComment"
			case "readSplatInMultiLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
			default:
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, fmt.Errorf("%d:%d: unexpected character %q (%s)", t.Line, t.Col, string(r), state)
			}
		case '%':
			switch state {
			case "inMultiLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
			case "inSingleLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
			case "readingText":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readingWhiteSpace":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readSplatInMultiLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
				state = "inMultiLineComment"
			case "start":
				for _, dirs := range []struct {
					command string
					id      int
				}{
					{"%default_type", SPECIAL_DIRECTIVE_DEFAULT_TYPE},
					{"%extra_context", SPECIAL_DIRECTIVE_EXTRA_CONTEXT},
					{"%syntax_error", SPECIAL_DIRECTIVE_SYNTAX_ERROR},
					{"%token_prefix", SPECIAL_DIRECTIVE_TOKEN_PREFIX},
					{"%token_type", SPECIAL_DIRECTIVE_TOKEN_TYPE},
				} {
					if compare(t.buf[t.pos:], dirs.command) {
						tok.ID = dirs.id
						t.pos, t.Col = t.pos+len(dirs.command), t.Col+len(dirs.command)
						tok.Value = t.buf[tok.Pos:t.pos]
						return &tok, nil
					}
				}
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, fmt.Errorf("%d:%d: unexpected character %q (%s)", t.Line, t.Col, string(r), state)
			}
		case '{':
			switch state {
			case "inMultiLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
			case "inSingleLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
			case "readingText":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readingWhiteSpace":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readSplatInMultiLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
				state = "inMultiLineComment"
			case "start":
				tok.ID = LEFT_CURLY_BRACE
				t.pos, t.Col = t.pos+w, t.Col+1
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			}
		case '}':
			switch state {
			case "inMultiLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
			case "inSingleLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
			case "readingText":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readingWhiteSpace":
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			case "readSplatInMultiLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
				state = "inMultiLineComment"
			case "start":
				tok.ID = RIGHT_CURLY_BRACE
				t.pos, t.Col = t.pos+w, t.Col+1
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, nil
			}
		default:
			switch state {
			case "inMultiLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
			case "inSingleLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
			case "readingText":
				if unicode.IsSpace(r) {
					tok.Value = t.buf[tok.Pos:t.pos]
					return &tok, nil
				}
				t.pos, t.Col = t.pos+w, t.Col+1
			case "readingWhiteSpace":
				if !unicode.IsSpace(r) {
					tok.Value = t.buf[tok.Pos:t.pos]
					return &tok, nil
				}
				t.pos, t.Col = t.pos+w, t.Col+1
			case "readSplatInMultiLineComment":
				t.pos, t.Col = t.pos+w, t.Col+1
				state = "inMultiLineComment"
			case "start":
				if unicode.IsLetter(r) {
					t.pos, t.Col = t.pos+w, t.Col+1
					state = "readingText"
				} else if unicode.IsSpace(r) {
					t.pos, t.Col = t.pos+w, t.Col+1
					state = "readingWhiteSpace"
				} else {
					tok.Value = t.buf[tok.Pos:t.pos]
					return &tok, fmt.Errorf("%d:%d: unexpected character %q (%s)", t.Line, t.Col, string(r), state)
				}
			default:
				tok.Value = t.buf[tok.Pos:t.pos]
				return &tok, fmt.Errorf("%d:%d: unexpected character %q (%s)", t.Line, t.Col, string(r), state)
			}
		}
	}
}

func compare(b []byte, s string) bool {
	return len(b) >= len(s) && bytes.Compare([]byte(s), b[:len(s)]) == 0
}
