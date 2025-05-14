// Package parser implements the LEMON LALR(1) parser generator in Go
// This is a Go port of the original C code from tool/lemon.c
package parser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	
	// Advanced options
	MakeHeaders    bool   // Output a makeheaders compatible file
	NoLineNos      bool   // Do not print #line statements
	PrintGrammar   bool   // Print grammar without actions
	PrintPreprocess bool  // Print input file after preprocessing
	SQL            bool   // Generate an SQLite3 table of parser statistics
	
	// Debug options
	Debug          bool   // Enable debug output during parser generation
	Trace          bool   // Enable trace output in the generated parser

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
	StateSet       *StateSet // The set of states in the LALR(1) state machine

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
	TemplateContent  string // Content of the template file
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

	// Set template file path and load template content
	var err error
	if p.TemplateFile != "" {
		// Use user-specified template file
		if filepath.IsAbs(p.TemplateFile) {
			p.TemplateFilename = p.TemplateFile
		} else {
			p.TemplateFilename = filepath.Join(filepath.Dir(grammarFile), p.TemplateFile)
		}

		// Read user template file
		templateData, err := os.ReadFile(p.TemplateFilename)
		if err != nil {
			return fmt.Errorf("error reading template file: %v", err)
		}
		p.TemplateContent = string(templateData)
	} else {
		// Use the embedded template
		p.TemplateContent, err = GetDefaultTemplate()
		if err != nil {
			return fmt.Errorf("error reading embedded template: %v", err)
		}
		p.TemplateFilename = "embedded lempar.c"
	}

	// Parse the grammar file
	err = p.parseGrammar()
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

	// Generate a makeheaders-compatible file if requested
	if p.MakeHeaders {
		err = p.generateMakeheadersFile()
		if err != nil {
			return fmt.Errorf("error generating makeheaders file: %v", err)
		}
	}

	// Generate SQLite3 output if requested
	if p.SQL {
		err = p.generateSQLFile()
		if err != nil {
			return fmt.Errorf("error generating SQL file: %v", err)
		}
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
	// Open the output file
	outputFile, err := os.Create(p.OutputFilename)
	if err != nil {
		return fmt.Errorf("cannot create output file: %v", err)
	}
	defer outputFile.Close()

	// Process the template
	output := p.processTemplate()

	// Write the output to the file
	_, err = outputFile.WriteString(output)
	if err != nil {
		return fmt.Errorf("error writing output file: %v", err)
	}

	return nil
}

// processTemplate replaces variables in the template with their actual values
func (p *Parser) processTemplate() string {
	// Make a copy of the template content
	result := p.TemplateContent

	// Replace %% markers with content
	splitPoints := []string{
		"/************ Begin %include sections from the grammar ************************/\n%%\n",
		"/**************** End token definitions ***************************************/\n\n%%\n",
		"/*********** Begin parsing tables **********************************************/\n%%\n",
		"%%\n};",
		"static const char *const yyTokenName[] = { \n%%\n};",
		"static const char *const yyRuleName[] = {\n%%\n};",
		"/********* Begin destructor definitions ***************************************/\n%%\n",
		"/******** Begin %stack_overflow code ******************************************/\n%%\n",
		"/* For rule J, yyRuleInfoLhs[J] contains the symbol on the left-hand side\n** of that rule */\nstatic const YYCODETYPE yyRuleInfoLhs[] = {\n%%\n};",
		"/* For rule J, yyRuleInfoNRhs[J] contains the negative of the number\n** of symbols on the right-hand side of that rule. */\nstatic const signed char yyRuleInfoNRhs[] = {\n%%\n};",
		"/********** Begin reduce actions **********************************************/\n%%\n",
		"/************ Begin %parse_failure code ***************************************/\n%%\n",
		"/************ Begin %syntax_error code ****************************************/\n%%\n",
		"/*********** Begin %parse_accept code *****************************************/\n%%\n",
	}

	// Content to replace the markers
	replacements := []string{
		p.generateIncludeCode(),       // %include sections
		p.generateDefines(),           // Control #defines
		p.generateParsingTables(),     // Parsing tables
		p.generateFallbacks(),         // Fallbacks
		p.generateTokenNames(),        // Token names
		p.generateRuleNames(),         // Rule names
		p.generateDestructors(),       // Destructors
		p.generateStackOverflow(),     // Stack overflow handling
		p.generateRuleInfoLhs(),       // Rule LHS info
		p.generateRuleInfoNRhs(),      // Rule RHS count info
		p.generateReduceActions(),     // Reduce actions
		p.generateParseFailure(),      // Parse failure code
		p.generateSyntaxError(),       // Syntax error code
		p.generateParseAccept(),       // Parse accept code
	}

	// Do the replacements
	for i, marker := range splitPoints {
		if i < len(replacements) {
			result = strings.Replace(result, marker, strings.Replace(marker, "%%\n", replacements[i]+"\n", 1), 1)
		}
	}

	// Replace P-a-r-s-e with the actual parser name
	result = strings.ReplaceAll(result, "Parse", p.Name)

	return result
}

// generateIncludeCode generates the include sections from the grammar
func (p *Parser) generateIncludeCode() string {
	return p.IncludeCode
}

// generateDefines generates the control #defines section
func (p *Parser) generateDefines() string {
	// This generates YYCODETYPE, YYNOCODE, etc.
	result := "/* These constants specify the various numeric values for terminal symbols\n"
	result += "** and nonterminal symbols, as well as the action codes used\n"
	result += "** in the action table */\n"
	
	// Add debugging/tracing support if requested
	if p.Debug {
		result += "#define LEMON_DEBUG 1\n"
	}
	
	if p.Trace {
		result += "#define LEMON_TRACE 1\n"
		result += "#define YYERRORSYMBOL " + fmt.Sprintf("%d\n", p.ErrorSym.Index)
		result += "#define YYWILDCARD " + fmt.Sprintf("%d\n", p.WildcardSym.Index)
	}
	
	return result
}

// generateFallbacks generates the fallback table
func (p *Parser) generateFallbacks() string {
	// This would list fallback tokens
	return "  0,  /* 0 = $ */\n"
}

// generateTokenNames generates the token name table
func (p *Parser) generateTokenNames() string {
	// Generate the token name table
	result := ""
	for _, sym := range p.Symbols {
		if sym.IsTerminal {
			result += fmt.Sprintf("  \"%s\",\n", sym.Name)
		}
	}
	return result
}

// generateRuleNames generates the rule name table
func (p *Parser) generateRuleNames() string {
	// Generate the rule name table
	result := ""
	for _, rule := range p.Rule {
		result += fmt.Sprintf("  \"%s ::= ", rule.Lhs.Name)
		for _, sym := range rule.Rhs {
			result += sym.Name + " "
		}
		result += "\",\n"
	}
	return result
}

// generateDestructors generates the destructor code
func (p *Parser) generateDestructors() string {
	// This would generate token and non-terminal destructors
	result := ""
	if p.TokenDestructor != "" {
		result += "/* Default destructor for terminals */\n"
		result += p.TokenDestructor + "\n"
	}
	if p.DefaultDestructor != "" {
		result += "/* Default destructor for non-terminals */\n"
		result += p.DefaultDestructor + "\n"
	}
	return result
}

// generateStackOverflow generates the stack overflow handling code
func (p *Parser) generateStackOverflow() string {
	if p.StackOverflow != "" {
		return p.StackOverflow
	}
	return ""
}

// generateRuleInfoLhs generates the rule left-hand side information
func (p *Parser) generateRuleInfoLhs() string {
	// Generate left-hand side information for each rule
	result := ""
	for _, rule := range p.Rule {
		result += fmt.Sprintf("  %d,  /* (%d) %s ::= */\n", rule.Lhs.Index, rule.RuleNum, rule.Lhs.Name)
	}
	return result
}

// generateRuleInfoNRhs generates the rule right-hand side count information
func (p *Parser) generateRuleInfoNRhs() string {
	// Generate right-hand side count information for each rule
	result := ""
	for _, rule := range p.Rule {
		result += fmt.Sprintf("  -%d,  /* (%d) %s ::= */\n", len(rule.Rhs), rule.RuleNum, rule.Lhs.Name)
	}
	return result
}

// generateReduceActions generates the reduce actions
func (p *Parser) generateReduceActions() string {
	// Generate code for reduce actions
	result := ""
	for i, rule := range p.Rule {
		result += fmt.Sprintf("  case %d: /* (%d) %s ::= ", i, rule.RuleNum, rule.Lhs.Name)
		for _, sym := range rule.Rhs {
			result += sym.Name + " "
		}
		result += "*/\n"
		result += "    break;\n"
	}
	return result
}

// generateParseFailure generates the parse failure code
func (p *Parser) generateParseFailure() string {
	if p.ParseFailure != "" {
		return p.ParseFailure
	}
	return ""
}

// generateSyntaxError generates the syntax error code
func (p *Parser) generateSyntaxError() string {
	if p.SyntaxError != "" {
		return p.SyntaxError
	}
	return ""
}

// generateParseAccept generates the parse accept code
func (p *Parser) generateParseAccept() string {
	if p.ParseAccept != "" {
		return p.ParseAccept
	}
	return ""
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
	// 1. Calculate the first sets for all non-terminals
	err := p.calculateFirstSets()
	if err != nil {
		return err
	}

	// 2. Find symbols that can derive epsilon (empty string)
	p.findNullableSymbols()

	// 3. Build the LALR(1) state machine
	err = p.buildStateMachine()
	if err != nil {
		return err
	}

	// 4. Check for conflicts and resolve if possible
	p.resolveConflicts()

	return nil
}

// calculateFirstSets computes the set of terminals that can appear at the
// beginning of any string derived from a non-terminal
func (p *Parser) calculateFirstSets() error {
	// Initialize first sets for all symbols
	for _, sym := range p.Symbols {
		// Terminal symbols can only derive themselves
		if sym.IsTerminal {
			sym.FirstSet = []*Symbol{sym}
		} else {
			sym.FirstSet = make([]*Symbol, 0)
		}
	}

	// Iteratively build the first sets until no more changes
	changed := true
	for changed {
		changed = false

		// For each rule X ::= Y1 Y2 ... Yn
		for _, rule := range p.Rule {
			lhs := rule.Lhs
			
			// If the rule has no RHS symbols, the LHS can derive epsilon
			if len(rule.Rhs) == 0 {
				lhs.Lambda = true
				continue
			}

			// Look at the first symbol on the RHS
			firstRHS := rule.Rhs[0]
			
			// Add all symbols from first(Y1) to first(X)
			oldSize := len(lhs.FirstSet)
			lhs.FirstSet = p.unionFirstSets(lhs.FirstSet, firstRHS.FirstSet)
			if len(lhs.FirstSet) > oldSize {
				changed = true
			}
			
			// If Y1 can derive epsilon, add first(Y2) to first(X), and so on
			i := 0
			for i < len(rule.Rhs) && rule.Rhs[i].Lambda {
				i++
				if i < len(rule.Rhs) {
					oldSize = len(lhs.FirstSet)
					lhs.FirstSet = p.unionFirstSets(lhs.FirstSet, rule.Rhs[i].FirstSet)
					if len(lhs.FirstSet) > oldSize {
						changed = true
					}
				}
			}
			
			// If all RHS symbols can derive epsilon, then LHS can derive epsilon
			if i == len(rule.Rhs) {
				lhs.Lambda = true
				changed = true
			}
		}
	}

	return nil
}

// unionFirstSets creates a union of two sets of symbols
func (p *Parser) unionFirstSets(set1, set2 []*Symbol) []*Symbol {
	// Create a map for quick lookup
	resultMap := make(map[*Symbol]bool)
	
	// Add all symbols from set1
	for _, sym := range set1 {
		resultMap[sym] = true
	}
	
	// Add all symbols from set2
	for _, sym := range set2 {
		resultMap[sym] = true
	}
	
	// Convert map back to slice
	result := make([]*Symbol, 0, len(resultMap))
	for sym := range resultMap {
		result = append(result, sym)
	}
	
	return result
}

// findNullableSymbols identifies symbols that can derive the empty string
func (p *Parser) findNullableSymbols() {
	// This has already been done during first set calculation
	// but we double-check here
	changed := true
	for changed {
		changed = false
		
		// Check each rule
		for _, rule := range p.Rule {
			// If the LHS is already nullable, skip
			if rule.Lhs.Lambda {
				continue
			}
			
			// Empty RHS means the LHS is nullable
			if len(rule.Rhs) == 0 {
				rule.Lhs.Lambda = true
				changed = true
				continue
			}
			
			// If all RHS symbols are nullable, the LHS is nullable
			allNullable := true
			for _, sym := range rule.Rhs {
				if !sym.Lambda {
					allNullable = false
					break
				}
			}
			
			if allNullable {
				rule.Lhs.Lambda = true
				changed = true
			}
		}
	}
}

// buildStateMachine constructs the LALR(1) state machine
func (p *Parser) buildStateMachine() error {
	// Initialize the state set
	p.StateSet = &StateSet{
		States: make([]*State, 0),
		NState: 0,
	}
	
	// Create the initial configurations (items) for the start state
	initConfigs := p.createInitialConfigurations()

	// Create the start state using the initial configurations
	startState := p.createNewState(p.StateSet, initConfigs)
	
	// Initialize the state stack with the start state
	stateStack := []*State{startState}

	// Process each state in the stack until no more states are added
	for len(stateStack) > 0 {
		// Pop a state from the stack
		state := stateStack[len(stateStack)-1]
		stateStack = stateStack[:len(stateStack)-1]
		
		// Get the transitions out of this state
		transitions := p.computeTransitions(state)
		
		// Create new states for each transition and add them to the stack
		for sym, configs := range transitions {
			// Check if this state already exists
			newState := p.findOrCreateState(p.StateSet, configs)
			
			// Add the state to the stack if it's new
			if newState.StateNum == p.StateSet.NState-1 {
				stateStack = append(stateStack, newState)
			}
			
			// Set the goto transition
			if state.Goto == nil {
				state.Goto = make(map[*Symbol]*State)
			}
			state.Goto[sym] = newState
		}
	}

	// Set the state count in the parser
	p.Nstate = p.StateSet.NState
	
	// Create the action tables for each state
	p.createActionTables(p.StateSet)

	return nil
}

// createInitialConfigurations creates the basis configurations for the start state
func (p *Parser) createInitialConfigurations() []*Configuration {
	configs := make([]*Configuration, 0)
	
	// Create a configuration for each rule where LHS is the start symbol
	for _, rule := range p.Rule {
		if rule.Lhs == p.StartSym {
			// Create a new configuration with the dot at the beginning
			config := &Configuration{
				Rp:        rule,
				Dot:       0,
				FollowSet: make([]int, 0),
				FwSet:     make([]int, 0),
				BasisFlag: true,
				RuleID:    rule.RuleNum,
			}
			
			// Add the configuration to the list
			configs = append(configs, config)
		}
	}

	// Compute the closure of the configurations
	configs = p.computeClosure(configs)

	return configs
}

// createNewState creates a new state from a set of configurations
func (p *Parser) createNewState(stateSet *StateSet, configs []*Configuration) *State {
	// Create the new state
	state := &State{
		Configs:      configs,
		BasisConfigs: p.findBasisConfigs(configs),
		StateNum:     stateSet.NState,
		Actions:      make([]*Action, 0),
		NTActions:    make([]*Action, 0),
		Goto:         make(map[*Symbol]*State),
	}

	// Add the state to the state set
	stateSet.States = append(stateSet.States, state)
	stateSet.NState++

	return state
}

// findBasisConfigs extracts the basis configurations from a set of configs
func (p *Parser) findBasisConfigs(configs []*Configuration) []*Configuration {
	basisConfigs := make([]*Configuration, 0)
	
	for _, config := range configs {
		if config.BasisFlag {
			basisConfigs = append(basisConfigs, config)
		}
	}
	
	return basisConfigs
}

// computeTransitions calculates the transitions from a state
func (p *Parser) computeTransitions(state *State) map[*Symbol][]*Configuration {
	transitions := make(map[*Symbol][]*Configuration)
	
	// Group configurations by the symbol after the dot
	for _, config := range state.Configs {
		// Skip if the dot is at the end of the rule
		if config.Dot >= len(config.Rp.Rhs) {
			continue
		}
		
		// Get the symbol after the dot
		sym := config.Rp.Rhs[config.Dot]
		
		// Create a new configuration with the dot advanced
		newConfig := &Configuration{
			Rp:        config.Rp,
			Dot:       config.Dot + 1,
			FollowSet: make([]int, len(config.FollowSet)),
			FwSet:     make([]int, len(config.FwSet)),
			BasisFlag: true,  // This is a basis config for the new state
			RuleID:    config.RuleID,
		}
		copy(newConfig.FollowSet, config.FollowSet)
		copy(newConfig.FwSet, config.FwSet)
		
		// Add the configuration to the appropriate transition
		if transitions[sym] == nil {
			transitions[sym] = make([]*Configuration, 0)
		}
		transitions[sym] = append(transitions[sym], newConfig)
	}
	
	// Compute closure for each transition
	for sym, configs := range transitions {
		transitions[sym] = p.computeClosure(configs)
	}
	
	return transitions
}

// computeClosure calculates the closure of a set of LR items
func (p *Parser) computeClosure(configs []*Configuration) []*Configuration {
	// Make a copy of the configurations to avoid modifying the input
	result := make([]*Configuration, len(configs))
	copy(result, configs)
	
	// Keep adding configurations until no more can be added
	changed := true
	for changed {
		changed = false
		oldSize := len(result)
		
		// Check each configuration
		for _, config := range result {
			// If the dot is at the end, skip
			if config.Dot >= len(config.Rp.Rhs) {
				continue
			}
			
			// Get the symbol after the dot
			sym := config.Rp.Rhs[config.Dot]
			
			// If it's a terminal, skip
			if sym.IsTerminal {
				continue
			}
			
			// For each rule with this non-terminal as LHS
			for _, rule := range p.Rule {
				if rule.Lhs != sym {
					continue
				}
				
				// Create a new configuration with the dot at the beginning
				newConfig := &Configuration{
					Rp:        rule,
					Dot:       0,
					FollowSet: make([]int, 0), // Will compute later
					FwSet:     make([]int, 0),  // Will compute later
					BasisFlag: false, // This is not a basis config
					RuleID:    rule.RuleNum,
				}
				
				// Check if this configuration already exists
				exists := false
				for _, existing := range result {
					if existing.Rp == newConfig.Rp && existing.Dot == newConfig.Dot {
						exists = true
						break
					}
				}
				
				// Add the configuration if it's new
				if !exists {
					result = append(result, newConfig)
				}
			}
		}
		
		// Check if we added new configurations
		if len(result) > oldSize {
			changed = true
		}
	}
	
	return result
}

// findOrCreateState finds an existing state or creates a new one
func (p *Parser) findOrCreateState(stateSet *StateSet, configs []*Configuration) *State {
	// Try to find an existing state with the same basis configurations
	for _, state := range stateSet.States {
		if p.sameConfigs(p.findBasisConfigs(configs), state.BasisConfigs) {
			// Found a matching state, return it
			return state
		}
	}
	
	// No matching state found, create a new one
	return p.createNewState(stateSet, configs)
}

// sameConfigs checks if two sets of configurations are the same
func (p *Parser) sameConfigs(configs1, configs2 []*Configuration) bool {
	if len(configs1) != len(configs2) {
		return false
	}
	
	// Create maps for quick lookups
	map1 := make(map[string]bool)
	map2 := make(map[string]bool)
	
	// Add all configs from the first set to map1
	for _, config := range configs1 {
		key := fmt.Sprintf("%d:%d", config.RuleID, config.Dot)
		map1[key] = true
	}
	
	// Add all configs from the second set to map2
	for _, config := range configs2 {
		key := fmt.Sprintf("%d:%d", config.RuleID, config.Dot)
		map2[key] = true
	}
	
	// Check if every key in map1 is also in map2
	for key := range map1 {
		if !map2[key] {
			return false
		}
	}
	
	// Check if every key in map2 is also in map1
	for key := range map2 {
		if !map1[key] {
			return false
		}
	}
	
	return true
}

// createActionTables creates the action tables for each state
func (p *Parser) createActionTables(stateSet *StateSet) {
	// For each state in the state machine
	for _, state := range stateSet.States {
		// Create action table for this state
		p.createActionsForState(state)
	}
}

// createActionsForState creates the action table for a single state
func (p *Parser) createActionsForState(state *State) {
	// Process each configuration in the state
	for _, config := range state.Configs {
		// If the dot is at the end, this is a reduce action
		if config.Dot >= len(config.Rp.Rhs) {
			// Create a reduce action for each symbol in the follow set
			for _, followIdx := range config.FollowSet {
				// Find the symbol with this index
				var followSym *Symbol
				for _, sym := range p.Symbols {
					if sym.Index == followIdx {
						followSym = sym
						break
					}
				}
				
				// Create the appropriate action
				if config.Rp.Lhs == p.StartSym && len(config.Rp.Rhs) == 1 && 
				   followSym == nil {
					// Accept action (start symbol and followSym is end-of-input)
					action := &Action{
						Sp:   nil, // End of input
						Type: ACCEPT,
						X:    0,
					}
					state.Actions = append(state.Actions, action)
				} else {
					// Regular reduce action
					action := &Action{
						Sp:   followSym,
						Type: REDUCE,
						X:    config.Rp.RuleNum,
					}
					state.Actions = append(state.Actions, action)
				}
			}
		} else {
			// The dot is not at the end, so this could be a shift action
			sym := config.Rp.Rhs[config.Dot]
			
			// If the symbol is a terminal and there's a goto transition
			if sym.IsTerminal && state.Goto[sym] != nil {
				// Create a shift action
				action := &Action{
					Sp:   sym,
					Type: SHIFT,
					X:    state.Goto[sym].StateNum,
				}
				state.Actions = append(state.Actions, action)
			}
			
			// If the symbol is a non-terminal and there's a goto transition
			if !sym.IsTerminal && state.Goto[sym] != nil {
				// Create a goto action
				action := &Action{
					Sp:   sym,
					Type: SHIFT, // Re-using SHIFT type for goto
					X:    state.Goto[sym].StateNum,
				}
				state.NTActions = append(state.NTActions, action)
			}
		}
	}
}

// resolveConflicts resolves shift-reduce and reduce-reduce conflicts
func (p *Parser) resolveConflicts() {
	// Initialize conflict counters
	shiftReduceCount := 0
	reduceReduceCount := 0
	resolvedCount := 0

	// If we don't have any states yet, return early
	if p.StateSet == nil || len(p.StateSet.States) == 0 {
		return
	}

	// Loop through all states
	for _, state := range p.StateSet.States {
		// Organize actions by symbol
		actions := make(map[*Symbol][]*Action)
		for _, action := range state.Actions {
			sp := action.Sp
			if actions[sp] == nil {
				actions[sp] = make([]*Action, 0)
			}
			actions[sp] = append(actions[sp], action)
		}

		// Look for symbols with multiple actions (conflicts)
		for sp, actList := range actions {
			if len(actList) <= 1 {
				continue // No conflict for this symbol
			}

			// We have a conflict for this symbol
			var shiftAct []*Action
			var reduceAct []*Action

			// Classify actions as shift or reduce
			for _, act := range actList {
				if act.Type == SHIFT {
					shiftAct = append(shiftAct, act)
				} else if act.Type == REDUCE {
					reduceAct = append(reduceAct, act)
				}
			}

			// Handle shift-reduce conflicts
			if len(shiftAct) > 0 && len(reduceAct) > 0 {
				shiftReduceCount += len(shiftAct) * len(reduceAct)
				
				// For each shift-reduce conflict
				for _, shift := range shiftAct {
					for i := 0; i < len(reduceAct); i++ {
						reduce := reduceAct[i]
						
						// Find the rule being reduced
						rule := p.Rule[reduce.X]
						
						// Check if the rule has a precedence
						rulePrec := -1
						if rule.Precedence != nil {
							rulePrec = rule.Precedence.Prec
						}
						
						// Get the token precedence
						tokenPrec := -1
						if sp != nil {
							tokenPrec = sp.Prec
						}
						
						// If both have precedence, we can resolve the conflict
						if rulePrec >= 0 && tokenPrec >= 0 {
							resolvedCount++
							
							// Decide based on precedence
							if rulePrec > tokenPrec {
								// Rule has higher precedence, remove the shift action
								p.removeAction(state, shift)
							} else if tokenPrec > rulePrec {
								// Token has higher precedence, remove the reduce action
								p.removeAction(state, reduce)
								reduceAct = append(reduceAct[:i], reduceAct[i+1:]...)
								i-- // Adjust the index after removal
							} else {
								// Same precedence, use associativity
								assoc := sp.Assoc
								if assoc == LEFT {
									// Left associativity -> reduce (remove shift)
									p.removeAction(state, shift)
								} else if assoc == RIGHT {
									// Right associativity -> shift (remove reduce)
									p.removeAction(state, reduce)
									reduceAct = append(reduceAct[:i], reduceAct[i+1:]...)
									i-- // Adjust the index after removal
								} else if assoc == NONASSOC {
									// Non-associative -> syntax error (remove both)
									p.removeAction(state, shift)
									p.removeAction(state, reduce)
									reduceAct = append(reduceAct[:i], reduceAct[i+1:]...)
									i-- // Adjust the index after removal
								}
							}
						}
					}
				}
			}

			// Handle reduce-reduce conflicts
			if len(reduceAct) > 1 {
				reduceReduceCount += len(reduceAct) - 1
				
				// Sort reduce actions by rule precedence and rule position
				p.sortReduceActions(reduceAct)
				
				// Keep only the first (highest precedence) reduction
				for i := 1; i < len(reduceAct); i++ {
					p.removeAction(state, reduceAct[i])
				}
			}
		}
	}

	// Print conflict resolution statistics if showing precedence conflicts
	if p.ShowPrecedence {
		fmt.Printf("\nConflicts resolved:\n")
		fmt.Printf("  %d shift-reduce conflicts\n", shiftReduceCount)
		fmt.Printf("  %d reduce-reduce conflicts\n", reduceReduceCount)
		fmt.Printf("  %d conflicts resolved using precedence rules\n", resolvedCount)
	}
}

// removeAction removes an action from a state's action list
func (p *Parser) removeAction(state *State, action *Action) {
	// Find and remove the action from the state's action list
	for i, act := range state.Actions {
		if act == action {
			state.Actions = append(state.Actions[:i], state.Actions[i+1:]...)
			break
		}
	}
}

// sortReduceActions sorts reduce actions by precedence and rule position
func (p *Parser) sortReduceActions(actions []*Action) {
	// Sort by precedence (highest first) and then by rule index (lowest first)
	sort.Slice(actions, func(i, j int) bool {
		rule1 := p.Rule[actions[i].X]
		rule2 := p.Rule[actions[j].X]
		
		// Get rule precedences
		prec1 := -1
		if rule1.Precedence != nil {
			prec1 = rule1.Precedence.Prec
		}
		prec2 := -1
		if rule2.Precedence != nil {
			prec2 = rule2.Precedence.Prec
		}
		
		// Compare by precedence
		if prec1 != prec2 {
			return prec1 > prec2 // Higher precedence first
		}
		
		// If same precedence, compare by rule index
		return rule1.Index < rule2.Index // Lower index first (earlier in grammar)
	})
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

// contains checks if an int slice contains a value
func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
