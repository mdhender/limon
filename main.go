package main

import (
	"flag"
	"fmt"
	"os"

	"limon/parser"
)

func main() {
	// Parse command-line flags similar to the original lemon tool
	var (
		// Basic options
		baseFlagPtr = flag.Bool("b", false, "Show only the basis in report")
		noCompressFlagPtr = flag.Bool("c", false, "Don't compress the action table")
		outputDirPtr = flag.String("d", "", "Output directory")
		showHelpPtr = flag.Bool("?", false, "Show help")
		showVersionPtr = flag.Bool("x", false, "Show version")
		statsFlagPtr = flag.Bool("s", false, "Show statistics about table generation")
		templateFilePtr = flag.String("T", "", "Specify a template file")
		
		// Advanced options
		definePtr = flag.String("D", "", "Define an %ifdef macro")
		makeheadersPtr = flag.Bool("m", false, "Output a makeheaders compatible file")
		noLineNosPtr = flag.Bool("l", false, "Do not print #line statements")
		printGrammarPtr = flag.Bool("g", false, "Print grammar without actions")
		printPreprocessPtr = flag.Bool("E", false, "Print input file after preprocessing")
		quietPtr = flag.Bool("q", false, "Don't print the report file")
		noResortPtr = flag.Bool("r", false, "Do not sort or renumber states")
		showPrecedencePtr = flag.Bool("p", false, "Show precedence levels in the report")
		sqlPtr = flag.Bool("S", false, "Generate an SQLite3 table of parser statistics")
		
		// Debug options
		debugPtr = flag.Bool("debug", false, "Enable debug output during parser generation")
		tracePtr = flag.Bool("trace", false, "Enable trace output in the generated parser")
	)

	flag.Parse()

	if *showHelpPtr {
		fmt.Println("Lemon (Go) LALR(1) Parser Generator")
		flag.PrintDefaults()
		return
	}

	if *showVersionPtr {
		fmt.Println("Lemon (Go) Parser Generator Version 0.1")
		return
	}

	// Check if a grammar file was specified
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Error: No grammar file specified")
		fmt.Println("Usage: limon [options] grammar-file")
		os.Exit(1)
	}

	grammarFile := args[0]
	
	// Create a new parser and process the grammar file
	p := parser.New()
	
	// Basic options
	p.Basisflag = *baseFlagPtr
	p.NoResort = *noCompressFlagPtr
	p.Stats = *statsFlagPtr
	p.TemplateFile = *templateFilePtr
	
	// Use the 'generated' directory by default to avoid cluttering with C files
	if *outputDirPtr == "" {
		p.Outdir = "generated"
	} else {
		p.Outdir = *outputDirPtr
	}
	
	// Advanced options
	if *definePtr != "" {
		// In the original Lemon, this defines a preprocessing macro
		// We'll store them and pass to our grammar preprocessor when implemented
		// For now we'll just print a warning
		fmt.Printf("Warning: -D option not fully implemented yet\n")
	}
	p.MakeHeaders = *makeheadersPtr
	p.NoLineNos = *noLineNosPtr
	p.PrintGrammar = *printGrammarPtr
	p.PrintPreprocess = *printPreprocessPtr
	p.Quiet = *quietPtr
	p.NoResort = *noResortPtr
	p.ShowPrecedence = *showPrecedencePtr
	p.SQL = *sqlPtr
	
	// Debug options
	p.Debug = *debugPtr
	p.Trace = *tracePtr
	err := p.GenerateParser(grammarFile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}