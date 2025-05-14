// Package parser implements the LEMON LALR(1) parser generator in Go
// This is a Go port of the original C code from tool/lemon.c
package parser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// Constants from the original C code
const (
	MAXRHS  = 1000 // Maximum number of symbols on the RHS of a rule
	Version = "1.0" // Parser generator version
)

// Parser represents the lemon parser generator
type Parser struct {
	// Parser configuration
	Basisflag      bool   // Output only basis configurations
	NoResort       bool   // Do not sort or renumber states
	ShowPrecedence bool   // Show precedence conflicts in the report
	Quiet          bool   // Don't print non-essential information
	Stats          bool   // Print performance statistics
	Grammar        string // Input grammar file name
	StartRule      string // Name of the start rule
	IncludePath    string // Directory for inclusion preprocessor
	Outdir         string // Directory where files are written
	TemplateFile   string // Template file

	// Parser state
	Nrule          int      // Number of rules
	Nsymbol        int      // Number of symbols
	Nstate         int      // Number of states
	Rule           []*Rule  // Array of rules
	Symbols        []*Symbol // Array of symbols
	StartSym       *Symbol // Start symbol
	TokenPrec      int     // Precedence for token symbols
	ErrorSym       *Symbol // The error symbol
	WildcardSym    *Symbol // The wildcard symbol
	Name           string  // Name of the generated parser
	TokenPrefix    string  // Prefix for token names
	TokenType      string  // Type of terminal symbols
	Vartype        string  // The default value of VARTYPE

	// Directive values
	IncludeCode       string // Code from %include directive
	ExtraCode         string // Code from %code directive
	ExtraArgument     string // Type from %extra_argument directive
	TokenDestructor   string // Code from %token_destructor directive
	DefaultDestructor string // Code from %default_destructor directive
	SyntaxError       string // Code from %syntax_error directive
	ParseAccept       string // Code from %parse_accept directive
	ParseFailure      string // Code from %parse_failure directive
	StackOverflow     string // Code from %stack_overflow directive
	StackSize         int    // Value from %stack_size directive (default: 100)

	// Path names
	TemplateFilename string // The template file name
	OutputFilename   string // The output file name
	HeaderFilename   string // The header file name
	ReportFilename   string // The report file name
}

// New creates a new Parser instance
func New() *Parser {
	return &Parser{
		TemplateFile: "lempar.c", // Default template file
		Rule:         make([]*Rule, 0),
		Symbols:      make([]*Symbol, 0),
		TokenType:    "void*", // Default token type
		Vartype:      "void*", // Default vartype
		StackSize:    100,     // Default stack size
	}
}

// GenerateParser converts a grammar file to a parser implementation
func (p *Parser) GenerateParser(grammarFile string) error {
	// Set the grammar file name
	p.Grammar = grammarFile

	// Set output file names based on the grammar file name
	baseFilename := strings.TrimSuffix(filepath.Base(grammarFile), filepath.Ext(grammarFile))
	outputDir := p.Outdir
	if outputDir == "" {
		outputDir = filepath.Dir(grammarFile)
	}

	p.OutputFilename = filepath.Join(outputDir, baseFilename+".c")
	p.HeaderFilename = filepath.Join(outputDir, baseFilename+".h")
	p.ReportFilename = filepath.Join(outputDir, baseFilename+".out")

	// Set template file path
	if filepath.IsAbs(p.TemplateFile) {
		p.TemplateFilename = p.TemplateFile
	} else {
		p.TemplateFilename = filepath.Join(filepath.Dir(grammarFile), p.TemplateFile)
	}

	// Parse the grammar file
	err := p.parseGrammar()
	if err != nil {
		return fmt.Errorf("error parsing grammar: %v", err)
	}

	// Set default parser name if not specified
	if p.Name == "" {
		p.Name = "Parse"
	}

	// Sort and analyze the grammar rules
	if !p.NoResort {
		p.sortRules()
	}

	// Generate the parser
	err = p.writeOutput()
	if err != nil {
		return fmt.Errorf("error generating output: %v", err)
	}

	// Print statistics if requested
	if p.Stats {
		fmt.Printf("\nParser statistics:\n")
		fmt.Printf("  %d terminals\n", p.countTerminals())
		fmt.Printf("  %d nonterminals\n", p.countNonterminals())
		fmt.Printf("  %d rules\n", p.Nrule)
		fmt.Printf("  %d symbols\n", p.Nsymbol)
	}

	return nil
}

// parseGrammar reads and parses the grammar file
func (p *Parser) parseGrammar() error {
	// Open the grammar file
	file, err := os.Open(p.Grammar)
	if err != nil {
		return fmt.Errorf("cannot open grammar file: %v", err)
	}
	defer file.Close()

	// Read the file line by line
	scanner := bufio.NewScanner(file)
	lineNo := 0
	var currentRule *Rule

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}

		// Handle directives
		if strings.HasPrefix(line, "%") {
			err := p.parseDirective(line, lineNo)
			if err != nil {
				return fmt.Errorf("line %d: %v", lineNo, err)
			}
			continue
		}

		// Handle grammar rules
		if strings.Contains(line, "::=") {
			currentRule, err = p.parseRule(line, lineNo)
			if err != nil {
				return fmt.Errorf("line %d: %v", lineNo, err)
			}
			p.Rule = append(p.Rule, currentRule)
			p.Nrule++
			continue
		}

		// Handle code blocks or continuations of rules
		// TODO: Implement code block parsing
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading grammar file: %v", err)
	}

	// After parsing, process the grammar
	return p.processGrammar()
}

// writeOutput generates the output files
func (p *Parser) writeOutput() error {
	// Generate the C source file
	err := p.generateParserCode()
	if err != nil {
		return fmt.Errorf("error generating parser code: %v", err)
	}

	// Generate the header file
	err = p.generateHeaderFile()
	if err != nil {
		return fmt.Errorf("error generating header file: %v", err)
	}

	// Generate the report file if not quiet
	if !p.Quiet {
		err = p.printReport()
		if err != nil {
			return fmt.Errorf("error generating report: %v", err)
		}
	}

	return nil
}

// generateParserCode generates the C parser implementation
func (p *Parser) generateParserCode() error {
	// Open the template file
	tpl, err := os.Open(p.TemplateFile)
	if err != nil {
		return fmt.Errorf("cannot open template file: %v", err)
	}
	defer tpl.Close()

	// Open the output file
	outputFile, err := os.Create(p.OutputFilename)
	if err != nil {
		return fmt.Errorf("cannot create output file: %v", err)
	}
	defer outputFile.Close()

	// TODO: Process template and generate parser

	return nil
}

// generateHeaderFile generates the header file with token definitions
func (p *Parser) generateHeaderFile() error {
	// Open the output file
	headerFile, err := os.Create(p.HeaderFilename)
	if err != nil {
		return fmt.Errorf("cannot create header file: %v", err)
	}
	defer headerFile.Close()

	// Generate header file content
	writer := bufio.NewWriter(headerFile)

	// Write header guards
	baseName := filepath.Base(p.HeaderFilename)
	guardName := strings.ToUpper(strings.ReplaceAll(baseName, ".", "_"))
	writer.WriteString(fmt.Sprintf("#ifndef %s\n", guardName))
	writer.WriteString(fmt.Sprintf("#define %s\n\n", guardName))

	// Write token definitions
	writer.WriteString("/* Token types */\n")
	for _, sym := range p.Symbols {
		if sym.IsTerminal {
			prefix := p.TokenPrefix
			writer.WriteString(fmt.Sprintf("#define %s%-30s %d\n", prefix, sym.Name, sym.Index))
		}
	}

	// Close the header guard
	writer.WriteString("\n#endif\n")

	return writer.Flush()
}

// printReport generates a report about the grammar
func (p *Parser) printReport() error {
	// Open the report file
	reportFile, err := os.Create(p.ReportFilename)
	if err != nil {
		return fmt.Errorf("cannot create report file: %v", err)
	}
	defer reportFile.Close()

	// Create a writer for the report file
	writer := bufio.NewWriter(reportFile)

	// Write report header
	writer.WriteString(fmt.Sprintf("Lemon Parser Generator - Grammar Report\n"))
	writer.WriteString(fmt.Sprintf("Grammar file: %s\n\n", p.Grammar))

	// Write grammar statistics
	writer.WriteString(fmt.Sprintf("Grammar Statistics:\n"))
	writer.WriteString(fmt.Sprintf("  Number of rules: %d\n", p.Nrule))
	writer.WriteString(fmt.Sprintf("  Number of symbols: %d\n", p.Nsymbol))
	writer.WriteString(fmt.Sprintf("  Number of terminals: %d\n", p.countTerminals()))
	writer.WriteString(fmt.Sprintf("  Number of non-terminals: %d\n\n", p.countNonterminals()))

	// Write symbol list
	writer.WriteString("Symbols:\n")
	for _, sym := range p.Symbols {
		if sym.IsTerminal {
			writer.WriteString(fmt.Sprintf("  Terminal %-20s (index: %d)\n", sym.Name, sym.Index))
		} else {
			writer.WriteString(fmt.Sprintf("  Non-terminal %-15s (index: %d)\n", sym.Name, sym.Index))
		}
	}
	writer.WriteString("\n")

	// Write rule list
	writer.WriteString("Rules:\n")
	for i, rule := range p.Rule {
		writer.WriteString(fmt.Sprintf("  %d: %s ::=", i, rule.Lhs.Name))
		for _, rhsSym := range rule.Rhs {
			writer.WriteString(fmt.Sprintf(" %s", rhsSym.Name))
		}
		writer.WriteString(".\n")
	}

	return writer.Flush()
}

// countTerminals returns the number of terminal symbols
func (p *Parser) countTerminals() int {
	count := 0
	for _, sym := range p.Symbols {
		if sym.IsTerminal {
			count++
		}
	}
	return count
}

// countNonterminals returns the number of non-terminal symbols
func (p *Parser) countNonterminals() int {
	count := 0
	for _, sym := range p.Symbols {
		if !sym.IsTerminal {
			count++
		}
	}
	return count
}

// sortRules sorts the rules by their precedence
func (p *Parser) sortRules() {
	// This is a simplification of the rule sorting in the C implementation
	// In a full implementation, we would sort rules based on precedence
	
	// For now, we just collect and renumber the rules
	rules := make([]*Rule, 0, p.Nrule)
	
	// Collect all rules
	for _, rule := range p.Rule {
		rules = append(rules, rule)
	}
	
	// Renumber rules
	for i, rule := range rules {
		rule.RuleNum = i
	}
	
	// Replace the rule list
	p.Rule = rules
}

// parseDirective processes a directive line starting with %
func (p *Parser) parseDirective(line string, lineNo int) error {
	// Extract the directive name
	directive := strings.TrimSpace(line)
	if len(directive) < 2 {
		return fmt.Errorf("empty directive")
	}

	// Remove the % prefix
	directive = directive[1:]

	// Split the directive into name and arguments
	parts := strings.SplitN(directive, " ", 2)
	directiveName := parts[0]
	directiveArgs := ""
	if len(parts) > 1 {
		directiveArgs = strings.TrimSpace(parts[1])
	}

	// Process the directive based on its name
	switch directiveName {
	case "token_type":
		return p.handleTokenType(directiveArgs)
	case "default_type":
		return p.handleDefaultType(directiveArgs)
	case "fallback":
		return p.handleFallback(directiveArgs)
	case "token":
		return p.handleToken(directiveArgs)
	case "type":
		return p.handleType(directiveArgs)
	case "left":
		return p.handlePrecedence(LEFT, directiveArgs)
	case "right":
		return p.handlePrecedence(RIGHT, directiveArgs)
	case "nonassoc":
		return p.handlePrecedence(NONASSOC, directiveArgs)
	case "start_symbol":
		return p.handleStartSymbol(directiveArgs)
	case "syntax_error":
		return p.handleSyntaxError(directiveArgs)
	case "parse_accept":
		return p.handleParseAccept(directiveArgs)
	case "parse_failure":
		return p.handleParseFailure(directiveArgs)
	case "stack_overflow":
		return p.handleStackOverflow(directiveArgs)
	case "extra_argument":
		return p.handleExtraArgument(directiveArgs)
	case "token_destructor":
		return p.handleTokenDestructor(directiveArgs)
	case "default_destructor":
		return p.handleDefaultDestructor(directiveArgs)
	case "destructor":
		return p.handleDestructor(directiveArgs)
	case "token_prefix":
		return p.handleTokenPrefix(directiveArgs)
	case "include":
		return p.handleInclude(directiveArgs)
	case "code":
		return p.handleCode(directiveArgs)
	case "name":
		return p.handleName(directiveArgs)
	case "stack_size":
		return p.handleStackSize(directiveArgs)
	case "wildcard":
		return p.handleWildcard(directiveArgs)
	default:
		return fmt.Errorf("unknown directive %%%s", directiveName)
	}
}

// handleTokenType processes the %token_type directive
func (p *Parser) handleTokenType(args string) error {
	// The argument should be a data type enclosed in {}
	if len(args) < 2 || args[0] != '{' || args[len(args)-1] != '}' {
		return fmt.Errorf("bad token_type directive syntax")
	}
	p.TokenType = args[1 : len(args)-1]
	return nil
}

// handleDefaultType processes the %default_type directive
func (p *Parser) handleDefaultType(args string) error {
	// The argument should be a data type enclosed in {}
	if len(args) < 2 || args[0] != '{' || args[len(args)-1] != '}' {
		return fmt.Errorf("bad default_type directive syntax")
	}
	p.Vartype = args[1 : len(args)-1]
	return nil
}

// handleFallback processes the %fallback directive
func (p *Parser) handleFallback(args string) error {
	// The fallback directive takes the form: %fallback ID TOKEN TOKEN...
	// First token is the fallback token, followed by tokens that fall back to it
	parts := strings.Split(args, ".")
	if len(parts) < 1 {
		return fmt.Errorf("missing period in fallback directive")
	}

	tokens := strings.Fields(parts[0])
	if len(tokens) < 2 {
		return fmt.Errorf("fallback directive needs at least two token names")
	}

	// Find or create the fallback token
	fallbackToken := p.findOrCreateSymbol(tokens[0])
	if !fallbackToken.IsTerminal {
		return fmt.Errorf("fallback token must be a terminal")
	}

	// Assign this fallback token to all specified tokens
	for i := 1; i < len(tokens); i++ {
		sym := p.findOrCreateSymbol(tokens[i])
		if !sym.IsTerminal {
			return fmt.Errorf("token '%s' in fallback directive must be a terminal", tokens[i])
		}
		sym.Fallback = fallbackToken
	}

	return nil
}

// handleToken processes the %token directive
func (p *Parser) handleToken(args string) error {
	// Extract token names until we hit a period
	parts := strings.Split(args, ".")
	if len(parts) < 1 {
		return fmt.Errorf("missing period in token directive")
	}
	
	tokens := strings.Fields(parts[0])
	for _, token := range tokens {
		// Create a new terminal symbol
		p.createTerminal(token)
	}
	
	return nil
}

// handleType processes the %type directive
func (p *Parser) handleType(args string) error {
	// The type directive takes the form: %type name {type}
	// It defines the data type of a non-terminal symbol
	parts := strings.SplitN(args, "{", 2)
	if len(parts) != 2 {
		return fmt.Errorf("missing type specification in type directive")
	}

	// Get the symbol name
	symbolName := strings.TrimSpace(parts[0])
	if symbolName == "" {
		return fmt.Errorf("missing symbol name in type directive")
	}

	// Get the type specification
	typeSpec := strings.TrimSpace(parts[1])
	if !strings.HasSuffix(typeSpec, "}") {
		return fmt.Errorf("missing closing brace in type specification")
	}
	typeSpec = typeSpec[:len(typeSpec)-1] // Remove the closing brace

	// Find or create the symbol
	symbol := p.findOrCreateSymbol(symbolName)
	if symbol.IsTerminal {
		return fmt.Errorf("%%type directive can only be applied to non-terminals")
	}

	// Set the symbol type
	symbol.Type = typeSpec

	return nil
}

// handlePrecedence processes the %left, %right, and %nonassoc directives
func (p *Parser) handlePrecedence(assoc int, args string) error {
	// Extract token names until we hit a period
	parts := strings.Split(args, ".")
	if len(parts) < 1 {
		return fmt.Errorf("missing period in precedence directive")
	}
	
	// Increment the precedence level
	p.TokenPrec++
	
	tokens := strings.Fields(parts[0])
	for _, token := range tokens {
		// Find or create the terminal and set its precedence
		sp := p.findOrCreateSymbol(token)
		if !sp.IsTerminal {
			return fmt.Errorf("%%left, %%right, and %%nonassoc apply to terminals only")
		}
		sp.Prec = p.TokenPrec
		sp.Assoc = assoc
	}
	
	return nil
}

// handleStartSymbol processes the %start_symbol directive
func (p *Parser) handleStartSymbol(args string) error {
	args = strings.TrimSpace(args)
	if args == "" {
		return fmt.Errorf("missing start symbol name")
	}
	p.StartRule = args
	return nil
}

// handleTokenPrefix processes the %token_prefix directive
func (p *Parser) handleTokenPrefix(args string) error {
	args = strings.TrimSpace(args)
	p.TokenPrefix = args
	return nil
}

// handleName processes the %name directive
func (p *Parser) handleName(args string) error {
	args = strings.TrimSpace(args)
	if args == "" {
		return fmt.Errorf("missing parser name")
	}
	p.Name = args
	return nil
}

// parseRule processes a grammar rule line containing ::=
func (p *Parser) parseRule(line string, lineNo int) (*Rule, error) {
	// Split the rule into LHS and RHS
	parts := strings.SplitN(line, "::=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed rule: missing ::=")
	}
	
	lhs := strings.TrimSpace(parts[0])
	rhs := strings.TrimSpace(parts[1])
	
	// Check if the RHS ends with a period
	hasCode := false
	// Variable for the code that might be attached to the rule (used later)
	// var code string
	parts = strings.SplitN(rhs, ".", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("malformed rule: missing period at end")
	}
	
	rhs = strings.TrimSpace(parts[0])
	
	// Check if there's code after the period
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		// We have code but we're not storing it yet
		// code = strings.TrimSpace(parts[1])
		hasCode = true
	}
	
	// Parse the LHS to get the symbol and any alias
	lhsSymbol, lhsAlias, err := p.parseSymbol(lhs)
	if err != nil {
		return nil, fmt.Errorf("error in LHS: %v", err)
	}
	
	// Create a new rule
	rule := &Rule{
		Lhs:        lhsSymbol,
		LhsAlias:   lhsAlias,
		LineNo:     lineNo,
		RuleNum:    p.Nrule,
		NoCode:     !hasCode,
	}
	
	// Parse the RHS to get all symbols
	rule.Rhs = make([]*Symbol, 0)
	rule.RhsAlias = make([]string, 0)
	
	// Split the RHS into tokens
	rhsTokens := strings.Fields(rhs)
	for _, token := range rhsTokens {
		symbol, alias, err := p.parseSymbol(token)
		if err != nil {
			return nil, fmt.Errorf("error in RHS symbol '%s': %v", token, err)
		}
		rule.Rhs = append(rule.Rhs, symbol)
		rule.RhsAlias = append(rule.RhsAlias, alias)
	}
	
	// Link the rule to its LHS symbol
	rule.NextLhs = lhsSymbol.Rule
	lhsSymbol.Rule = rule
	
	return rule, nil
}

// parseSymbol parses a symbol name and any alias it might have
func (p *Parser) parseSymbol(name string) (*Symbol, string, error) {
	// Check if there's an alias in parentheses
	alias := ""
	nameParts := strings.SplitN(name, "(", 2)
	if len(nameParts) > 1 {
		// Extract the alias from between parentheses
		name = strings.TrimSpace(nameParts[0])
		aliasPart := nameParts[1]
		if !strings.HasSuffix(aliasPart, ")") {
			return nil, "", fmt.Errorf("missing closing parenthesis in alias")
		}
		alias = aliasPart[:len(aliasPart)-1]
	}
	
	// Find or create the symbol
	symbol := p.findOrCreateSymbol(name)
	return symbol, alias, nil
}

// findOrCreateSymbol finds a symbol by name or creates a new one
func (p *Parser) findOrCreateSymbol(name string) *Symbol {
	// Check if the symbol already exists
	for _, sym := range p.Symbols {
		if sym.Name == name {
			return sym
		}
	}
	
	// Create a new symbol
	newSym := &Symbol{
		Name:       name,
		Index:      p.Nsymbol,
		IsTerminal: isUpper(rune(name[0])),
		Prec:       -1, // No precedence initially
	}
	
	p.Symbols = append(p.Symbols, newSym)
	p.Nsymbol++
	
	return newSym
}

// createTerminal creates a new terminal symbol
func (p *Parser) createTerminal(name string) *Symbol {
	// Make sure the first character is uppercase (terminal symbol)
	if len(name) == 0 || !isUpper(rune(name[0])) {
		// Convert to uppercase or return nil
		if len(name) > 0 {
			name = strings.ToUpper(name[:1]) + name[1:]
		} else {
			return nil
		}
	}
	
	// Find or create the symbol
	symbol := p.findOrCreateSymbol(name)
	return symbol
}

// processGrammar does post-processing after parsing the grammar
func (p *Parser) processGrammar() error {
	// Set the start symbol if it wasn't specified with %start_symbol
	if p.StartRule != "" {
		// Find the start symbol
		for _, sym := range p.Symbols {
			if sym.Name == p.StartRule {
				p.StartSym = sym
				break
			}
		}
		if p.StartSym == nil {
			return fmt.Errorf("start symbol %s not found", p.StartRule)
		}
	} else if p.Nrule > 0 {
		// Use the left-hand side of the first rule
		p.StartSym = p.Rule[0].Lhs
	}
	
	// Create error and wildcard symbols if needed
	p.ErrorSym = p.findOrCreateSymbol("error")
	
	// Analyze the grammar
	return p.analyzeGrammar()
}

// analyzeGrammar performs grammar analysis
func (p *Parser) analyzeGrammar() error {
	// TODO: Implement grammar analysis
	return nil
}

// handleSyntaxError processes the %syntax_error directive
func (p *Parser) handleSyntaxError(args string) error {
	// The syntax_error directive takes the form: %syntax_error { C code }
	// It specifies code to run when a syntax error occurs
	
	// Check if the argument starts with an opening brace
	if !strings.HasPrefix(args, "{") {
		return fmt.Errorf("syntax_error directive requires a code block in braces")
	}
	
	// Find the matching closing brace
	// This is a simplified approach; a proper implementation would handle nested braces
	if !strings.HasSuffix(args, "}") {
		return fmt.Errorf("missing closing brace in syntax_error directive")
	}
	
	// Store the code block for later use when generating the parser
	p.SyntaxError = args[1:len(args)-1] // Remove braces
	
	return nil
}

// handleParseAccept processes the %parse_accept directive
func (p *Parser) handleParseAccept(args string) error {
	// The parse_accept directive takes the form: %parse_accept { C code }
	// It specifies code to run when the parser accepts its input
	
	// Check if the argument starts with an opening brace
	if !strings.HasPrefix(args, "{") {
		return fmt.Errorf("parse_accept directive requires a code block in braces")
	}
	
	// Find the matching closing brace
	if !strings.HasSuffix(args, "}") {
		return fmt.Errorf("missing closing brace in parse_accept directive")
	}
	
	// Store the code block for later use when generating the parser
	p.ParseAccept = args[1:len(args)-1] // Remove braces
	
	return nil
}

// handleParseFailure processes the %parse_failure directive
func (p *Parser) handleParseFailure(args string) error {
	// The parse_failure directive takes the form: %parse_failure { C code }
	// It specifies code to run when the parser fails to parse the input
	
	// Check if the argument starts with an opening brace
	if !strings.HasPrefix(args, "{") {
		return fmt.Errorf("parse_failure directive requires a code block in braces")
	}
	
	// Find the matching closing brace
	if !strings.HasSuffix(args, "}") {
		return fmt.Errorf("missing closing brace in parse_failure directive")
	}
	
	// Store the code block for later use when generating the parser
	p.ParseFailure = args[1:len(args)-1] // Remove braces
	
	return nil
}

// handleStackOverflow processes the %stack_overflow directive
func (p *Parser) handleStackOverflow(args string) error {
	// The stack_overflow directive takes the form: %stack_overflow { C code }
	// It specifies code to run when the parser stack overflows
	
	// Check if the argument starts with an opening brace
	if !strings.HasPrefix(args, "{") {
		return fmt.Errorf("stack_overflow directive requires a code block in braces")
	}
	
	// Find the matching closing brace
	if !strings.HasSuffix(args, "}") {
		return fmt.Errorf("missing closing brace in stack_overflow directive")
	}
	
	// Store the code block for later use when generating the parser
	p.StackOverflow = args[1:len(args)-1] // Remove braces
	
	return nil
}

// handleExtraArgument processes the %extra_argument directive
func (p *Parser) handleExtraArgument(args string) error {
	// The extra_argument directive takes the form: %extra_argument { type name }
	// It adds an extra argument to the Parse() function
	
	// Check if the argument starts with an opening brace
	if !strings.HasPrefix(args, "{") {
		return fmt.Errorf("extra_argument directive requires a type specification in braces")
	}
	
	// Find the matching closing brace
	if !strings.HasSuffix(args, "}") {
		return fmt.Errorf("missing closing brace in extra_argument directive")
	}
	
	// Extract the type specification
	typeSpec := args[1:len(args)-1] // Remove braces
	typeSpec = strings.TrimSpace(typeSpec)
	
	// Store the extra argument type specification
	p.ExtraArgument = typeSpec
	
	return nil
}

// handleTokenDestructor processes the %token_destructor directive
func (p *Parser) handleTokenDestructor(args string) error {
	// The token_destructor directive takes the form: %token_destructor { C code }
	// It specifies a destructor for all terminal symbols
	
	// Check if the argument starts with an opening brace
	if !strings.HasPrefix(args, "{") {
		return fmt.Errorf("token_destructor directive requires a code block in braces")
	}
	
	// Find the matching closing brace
	if !strings.HasSuffix(args, "}") {
		return fmt.Errorf("missing closing brace in token_destructor directive")
	}
	
	// Store the code block for later use when generating the parser
	p.TokenDestructor = args[1:len(args)-1] // Remove braces
	
	return nil
}

// handleDefaultDestructor processes the %default_destructor directive
func (p *Parser) handleDefaultDestructor(args string) error {
	// The default_destructor directive takes the form: %default_destructor { C code }
	// It specifies a default destructor for all non-terminal symbols
	
	// Check if the argument starts with an opening brace
	if !strings.HasPrefix(args, "{") {
		return fmt.Errorf("default_destructor directive requires a code block in braces")
	}
	
	// Find the matching closing brace
	if !strings.HasSuffix(args, "}") {
		return fmt.Errorf("missing closing brace in default_destructor directive")
	}
	
	// Store the code block for later use when generating the parser
	p.DefaultDestructor = args[1:len(args)-1] // Remove braces
	
	return nil
}

// handleDestructor processes the %destructor directive
func (p *Parser) handleDestructor(args string) error {
	// The destructor directive takes the form: %destructor symbol { C code }
	// It specifies a destructor for a specific non-terminal symbol
	
	// Split the args into symbol name and code block
	parts := strings.SplitN(args, "{", 2)
	if len(parts) != 2 {
		return fmt.Errorf("destructor directive requires a symbol name and a code block")
	}
	
	// Extract the symbol name
	symbolName := strings.TrimSpace(parts[0])
	if symbolName == "" {
		return fmt.Errorf("missing symbol name in destructor directive")
	}
	
	// Extract the code block
	codeBlock := parts[1]
	if !strings.HasSuffix(codeBlock, "}") {
		return fmt.Errorf("missing closing brace in destructor directive")
	}
	
	// Find the symbol
	symbol := p.findOrCreateSymbol(symbolName)
	if symbol.IsTerminal {
		return fmt.Errorf("destructor directive can only be applied to non-terminals")
	}
	
	// Store the destructor for the symbol
	symbol.Destructor = codeBlock[:len(codeBlock)-1] // Remove the closing brace
	
	return nil
}

// handleInclude processes the %include directive
func (p *Parser) handleInclude(args string) error {
	// The include directive takes the form: %include { C code }
	// It specifies C code to be included at the top of the generated parser

	// Check if the argument starts with an opening brace
	if !strings.HasPrefix(args, "{") {
		return fmt.Errorf("include directive requires a code block in braces")
	}

	// Find the matching closing brace
	if !strings.HasSuffix(args, "}") {
		return fmt.Errorf("missing closing brace in include directive")
	}

	// Extract and store the include code
	p.IncludeCode = args[1:len(args)-1] // Remove braces
	
	return nil
}

// handleCode processes the %code directive
func (p *Parser) handleCode(args string) error {
	// The code directive takes the form: %code { C code }
	// It specifies C code to be included at the end of the generated parser

	// Check if the argument starts with an opening brace
	if !strings.HasPrefix(args, "{") {
		return fmt.Errorf("code directive requires a code block in braces")
	}

	// Find the matching closing brace
	if !strings.HasSuffix(args, "}") {
		return fmt.Errorf("missing closing brace in code directive")
	}

	// Extract and store the code
	p.ExtraCode = args[1:len(args)-1] // Remove braces
	
	return nil
}

// handleStackSize processes the %stack_size directive
func (p *Parser) handleStackSize(args string) error {
	// The stack_size directive takes the form: %stack_size number
	// It specifies the maximum size of the parser stack

	// Trim whitespace
	args = strings.TrimSpace(args)
	if args == "" {
		return fmt.Errorf("stack_size directive requires a number")
	}

	// Parse the stack size value
	stackSize, err := strconv.Atoi(args)
	if err != nil {
		return fmt.Errorf("invalid stack size value: %v", err)
	}

	// Check if the stack size is valid
	if stackSize <= 0 {
		return fmt.Errorf("stack size must be a positive integer")
	}

	// Store the stack size for use when generating the parser
	p.StackSize = stackSize
	
	return nil
}

// handleWildcard processes the %wildcard directive
func (p *Parser) handleWildcard(args string) error {
	// The wildcard directive takes the form: %wildcard TOKEN.
	// It specifies a token that matches any input token

	// Extract token name until we hit a period
	parts := strings.Split(args, ".")
	if len(parts) < 1 {
		return fmt.Errorf("missing period in wildcard directive")
	}

	// Get the token name
	tokenName := strings.TrimSpace(parts[0])
	if tokenName == "" {
		return fmt.Errorf("missing token name in wildcard directive")
	}

	// Find or create the token
	wildcardToken := p.createTerminal(tokenName)
	if wildcardToken == nil {
		return fmt.Errorf("invalid wildcard token name")
	}

	// Set the wildcard token
	p.WildcardSym = wildcardToken

	return nil
}

// Helper functions to match the C macros
func isSpace(r rune) bool {
	return unicode.IsSpace(r)
}

func isDigit(r rune) bool {
	return unicode.IsDigit(r)
}

func isAlnum(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isAlpha(r rune) bool {
	return unicode.IsLetter(r)
}

func isUpper(r rune) bool {
	return unicode.IsUpper(r)
}

func isLower(r rune) bool {
	return unicode.IsLower(r)
}