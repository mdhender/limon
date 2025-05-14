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

package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/mdhender/limon/parser"
	"github.com/mdhender/limon/scanner"
)

func main() {
	filename := "testdata/sqlite.lemon"
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	var s scanner.Scanner
	s.Init(buf)
	var tok rune
	for tok != scanner.EOF {
		tok = s.Scan()
		fmt.Printf("%s: %s: %q\n", s.Pos(), scanner.TokenString(tok), s.TokenText())
	}

	// tree, err := ParseFile("testdata/sqlite.lemon")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(tree)
}

// ParseFile will parse an input file, returning the resulting tree or the first error encountered.
func ParseFile(filename string) (*ParseTree, error) {
	tokenizer, err := scanner.Create(filename)
	if err != nil {
		return nil, err
	}

	parser, err := parser.Alloc()
	if err != nil {
		return nil, err
	}

	state := InitMyParserState()

	token, err := tokenizer.GetNextToken()
	for ; err == nil && token.ID != scanner.EOF; token, err = tokenizer.GetNextToken() {
		fmt.Printf("%d:%d: %d: %q\n", token.Line, token.Col, token.ID, string(token.Value))
		state.Line, state.Col = token.Line, token.Col
		parser.Parse(token.ID, token.Value, state)
	}
	if err != nil {
		return nil, err
	}

	if err = parser.Parse(scanner.EOF, nil, state); err != nil {
		return nil, err
	}

	parser.Free()
	tokenizer.Free()

	return state.Root, nil
}

// ParseTree is the generated tree.
type ParseTree struct{}

// MyParserState holds state that the parser will pass blindly through.
// (Must be defined by the %extra_argument directive!)
type MyParserState struct {
	Line, Col int
	Root      *ParseTree
}

// InitMyParserState returns an initialized state to use for our purposes when parsing.
func InitMyParserState() *MyParserState {
	return &MyParserState{}
}
