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

// Package parser implements the parser and the parser generator?
// Is that right?
// Or does this somehow consume the output of the generator?
package parser

// Parser holds the current state of the parser.
type Parser struct{}

// Alloc returns an initialized parser.
func Alloc() (*Parser, error) {
	return &Parser{}, nil
}

// Free releases all memory used by the parser.
func (p *Parser) Free() {
}

// Parse does something?
//
// It expects to be called in a loop like:
//   token, err := tokenizer.GetNextToken()
//   for token != nil && err != nil {
// 	    state.Line, state.Col = token.Line, token.Col
// 	    parser.Parse(token.ID, token.Value, state)
// 	    token, err = tokenizer.GetNextToken()
//   }
//
// Note that somehow an ID of zero means end of input.
func (p *Parser) Parse(id int, data []byte, state interface{}) error {
	return nil
}
