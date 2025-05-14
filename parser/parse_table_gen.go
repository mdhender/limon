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

	// Generate the parsing tables
	result := "/* The action table */\n"
	result += "static const YYACTIONTYPE yy_action[] = {\n"

	// Determine the number of symbols and states
	nStates := p.StateSet.NState
	nSymbols := p.Nsymbol
	nTerminals := p.countTerminals()

	// Create action and goto tables
	actionTable := make([][]int, nStates)
	gotoTable := make([][]int, nStates)

	// Initialize tables with error/no-action values
	for i := 0; i < nStates; i++ {
		actionTable[i] = make([]int, nTerminals)
		gotoTable[i] = make([]int, nSymbols-nTerminals)

		// Fill with -1 (error/no action)
		for j := 0; j < nTerminals; j++ {
			actionTable[i][j] = -1
		}
		for j := 0; j < nSymbols-nTerminals; j++ {
			gotoTable[i][j] = -1
		}
	}

	// Fill the tables by iterating through states and their actions
	for _, state := range p.StateSet.States {
		stateNum := state.StateNum

		// Process terminal symbol actions (shift, reduce, accept, error)
		for _, action := range state.Actions {
			// Skip actions for non-terminals
			if !action.Sp.IsTerminal {
				continue
			}

			symbolIndex := action.Sp.Index

			// Encode action type and value
			var actionCode int
			switch action.Type {
			case SHIFT:
				// Shift actions are encoded as positive state numbers
				actionCode = action.X // The state to shift to
			case REDUCE:
				// Reduce actions are encoded as negative rule numbers
				actionCode = -action.X - 1 // The rule to reduce by (-1 to avoid conflict with ERROR)
			case ACCEPT:
				// Accept is a special code
				actionCode = -9999 // Use a very negative number for accept
			case ERROR:
				// Error remains -1
				actionCode = -1
			}

			// Store action in the table
			actionTable[stateNum][symbolIndex] = actionCode
		}

		// Process non-terminal actions (goto)
		for sym, gotoState := range state.Goto {
			// Skip terminal symbols
			if sym.IsTerminal {
				continue
			}

			// Non-terminal symbols have their own indices
			nontermIndex := sym.Index - nTerminals
			if nontermIndex >= 0 {
				gotoTable[stateNum][nontermIndex] = gotoState.StateNum
			}
		}
	}

	// Build a combined action and goto table as required by the Lemon template
	actionsArr := make([]int, 0, nStates*(nTerminals+nSymbols-nTerminals+2))

	// For each state, we need:
	// 1. The base offset for the action table entries
	// 2. The base offset for the goto table entries
	// 3. The number of terminal entries
	// 4. The number of non-terminal entries
	actOffsets := make([]int, nStates)
	gotoOffsets := make([]int, nStates)

	// Track the current offset as we build the table
	offset := 0

	// For each state, add its action and goto entries
	for stateNum := 0; stateNum < nStates; stateNum++ {
		// Record the action offset for this state
		actOffsets[stateNum] = offset

		// Add actions for terminals
		actionsInState := 0
		for termIndex := 0; termIndex < nTerminals; termIndex++ {
			act := actionTable[stateNum][termIndex]
			if act != -1 {
				actionsArr = append(actionsArr, termIndex) // Symbol index
				actionsArr = append(actionsArr, act)       // Action code
				actionsInState++
				offset += 2
			}
		}

		// Record the goto offset for this state
		gotoOffsets[stateNum] = offset

		// Add gotos for non-terminals
		for nontermIndex := 0; nontermIndex < nSymbols-nTerminals; nontermIndex++ {
			gotoVal := gotoTable[stateNum][nontermIndex]
			if gotoVal != -1 {
				actionsArr = append(actionsArr, nontermIndex+nTerminals) // Symbol index
				actionsArr = append(actionsArr, gotoVal)                // State to go to
				offset += 2
			}
		}
	}

	// Generate the action table as C code
	for i, action := range actionsArr {
		if i%8 == 0 {
			result += "  "
		}
		result += fmt.Sprintf("%d, ", action)
		if i%8 == 7 || i == len(actionsArr)-1 {
			result += fmt.Sprintf(" /* %d */\n", i-i%8)
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

	// Extract lookaheads from configurations
	lookaheads := make([]int, 0)
	for _, state := range p.StateSet.States {
		for _, config := range state.Configs {
			for _, la := range config.FollowSet {
				if !contains(lookaheads, la) {
					lookaheads = append(lookaheads, la)
				}
			}
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

	return result
}