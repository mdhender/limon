package parser

import (
	"fmt"
	"sort"
)

// generateParsingTables generates the parsing tables
func (p *Parser) generateParsingTables() string {
	// Ensure we have a state machine to generate tables from
	if p.StateSet == nil || len(p.StateSet.States) == 0 {
		return "/* No states in the state machine */\n"
	}

	// Generate the parsing tables using compression
	result := "/* The action table */\n"
	result += "static const YYACTIONTYPE yy_action[] = {\n"

	// Determine the number of symbols and states
	nStates := p.StateSet.NState
	nSymbols := p.Nsymbol
	nTerminals := p.countTerminals()

	// Compress the tables
	actionsTable, actOffsets, gotoOffsets := p.compressActionTable()

	// Generate the action table as C code
	for i, entry := range actionsTable {
		if i%4 == 0 {
			result += "  "
		}
		value := 0
		if entry.Lookahead >= 0 {
			value = entry.Action
		}
		result += fmt.Sprintf("%d, ", value)
		if i%4 == 3 || i == len(actionsTable)-1 {
			result += fmt.Sprintf(" /* %d */\n", i-i%4)
		}
	}
	result += "};\n\n"

	// Generate action offset table
	result += "/* The action offset table */\n"
	result += "static const YYCODETYPE yy_action_offset[] = {\n"
	for i, offset := range actOffsets {
		if i%8 == 0 {
			result += "  "
		}
		result += fmt.Sprintf("%d, ", offset)
		if i%8 == 7 || i == len(actOffsets)-1 {
			result += fmt.Sprintf(" /* %d */\n", i-i%8)
		}
	}
	result += "};\n\n"

	// Generate goto offset table
	result += "/* The goto offset table */\n"
	result += "static const YYCODETYPE yy_goto_offset[] = {\n"
	for i, offset := range gotoOffsets {
		if i%8 == 0 {
			result += "  "
		}
		result += fmt.Sprintf("%d, ", offset)
		if i%8 == 7 || i == len(gotoOffsets)-1 {
			result += fmt.Sprintf(" /* %d */\n", i-i%8)
		}
	}
	result += "};\n\n"

	// Generate lookahead table
	result += "/* The LALR(1) lookahead sets */\n"
	result += "static const YYCODETYPE yy_lookahead[] = {\n"

	// Extract lookaheads from action table entries
	lookaheads := make([]int, 0, len(actionsTable))
	for _, entry := range actionsTable {
		if entry.Lookahead >= 0 {
			lookaheads = append(lookaheads, entry.Lookahead)
		}
	}

	// Sort lookaheads for deterministic output
	sort.Ints(lookaheads)
	
	// Output the lookahead array
	for i, la := range lookaheads {
		if i%8 == 0 {
			result += "  "
		}
		result += fmt.Sprintf("%d, ", la)
		if i%8 == 7 || i == len(lookaheads)-1 {
			result += fmt.Sprintf(" /* %d */\n", i-i%8)
		}
	}
	result += "};\n\n"

	// Add metadata about the tables
	result += fmt.Sprintf("#define YY_SHIFT_USE_DFLT %d\n", -1)
	result += fmt.Sprintf("#define YY_SHIFT_COUNT %d\n", nStates)
	result += fmt.Sprintf("#define YY_SHIFT_MIN %d\n", 0)
	result += fmt.Sprintf("#define YY_SHIFT_MAX %d\n", nStates-1)
	result += fmt.Sprintf("#define YY_REDUCE_USE_DFLT %d\n", -1)
	result += fmt.Sprintf("#define YY_REDUCE_COUNT %d\n", p.Nrule)
	result += fmt.Sprintf("#define YY_REDUCE_MIN %d\n", 0)
	result += fmt.Sprintf("#define YY_REDUCE_MAX %d\n", p.Nrule-1)
	result += fmt.Sprintf("#define YYERRORSYMBOL %d\n", p.ErrorSym.Index)
	result += fmt.Sprintf("#define YYNOCODE %d\n", nSymbols)

	// Print statistics about compression if Stats is true
	if p.Stats {
		origSize := nStates * (nTerminals + nSymbols - nTerminals)
		compSize := len(actionsTable)
		compRatio := float64(compSize) / float64(origSize)
		fmt.Printf("Table compression: %d bytes to %d bytes (%.2f%%)\n", 
			origSize, compSize, 100*compRatio)
	}

	return result
}