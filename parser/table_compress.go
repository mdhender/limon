package parser

import (
)

// Constants for action table entry types
const (
	NOT_USED = -1 // Used to mark deleted entries during compression
)

// A transaction set represents a group of actions to be inserted into the table
type transactionSet struct {
	lookaheads []int   // Token lookaheads
	actions    []int   // Actions to take for each lookahead
	minLookahead int   // Minimum lookahead value in the set
	maxLookahead int   // Maximum lookahead value in the set
	defaultAction int  // Default action for this transaction set
}

// ActionTableEntry represents an entry in the action table
type ActionTableEntry struct {
	Lookahead int  // Token to look for (-1 if this slot is empty)
	Action    int  // Action to take
}

// ActionTable represents a compressed action table
type ActionTable struct {
	actions       []ActionTableEntry  // The action table entries
	nAction       int                 // Number of used slots in the action table
	nterminal     int                 // Number of terminal symbols
	nsymbol       int                 // Total number of symbols
}

// NewActionTable creates a new action table with initial capacity
func NewActionTable(nsymbol, nterminal int) *ActionTable {
	// Allocate initial table with a reasonable size
	initialSize := nsymbol * 2
	actions := make([]ActionTableEntry, initialSize)
	
	// Initialize all entries as empty
	for i := 0; i < initialSize; i++ {
		actions[i].Lookahead = NOT_USED
		actions[i].Action = NOT_USED
	}
	
	return &ActionTable{
		actions:   actions,
		nAction:   0,
		nterminal: nterminal,
		nsymbol:   nsymbol,
	}
}

// CreateTransaction creates a new transaction set for a state
func CreateTransaction() *transactionSet {
	return &transactionSet{
		lookaheads:    make([]int, 0),
		actions:       make([]int, 0),
		minLookahead:  0,
		maxLookahead:  0,
		defaultAction: NOT_USED,
	}
}

// AddAction adds an action for a specific lookahead to the transaction set
func (ts *transactionSet) AddAction(lookahead, action int) {
	// Update min/max lookahead values
	if len(ts.lookaheads) == 0 {
		ts.minLookahead = lookahead
		ts.maxLookahead = lookahead
	} else {
		if lookahead < ts.minLookahead {
			ts.minLookahead = lookahead
		}
		if lookahead > ts.maxLookahead {
			ts.maxLookahead = lookahead
		}
	}
	
	// Add the lookahead and action
	ts.lookaheads = append(ts.lookaheads, lookahead)
	ts.actions = append(ts.actions, action)
}

// FindDefaultAction determines if there's a common action that can be used as default
func (ts *transactionSet) FindDefaultAction() bool {
	if len(ts.lookaheads) == 0 {
		return false
	}
	
	// Count occurrences of each action
	actionCounts := make(map[int]int)
	for _, action := range ts.actions {
		actionCounts[action]++
	}
	
	// Find the most common action
	bestAction := ts.actions[0]
	bestCount := actionCounts[bestAction]
	
	for action, count := range actionCounts {
		if count > bestCount {
			bestAction = action
			bestCount = count
		}
	}
	
	// If the best action occurs multiple times, use it as default
	if bestCount > 1 {
		ts.defaultAction = bestAction
		return true
	}
	
	return false
}

// Insert adds a transaction set to the action table, finding the best offset
// Returns the offset where the transaction was inserted
func (at *ActionTable) Insert(ts *transactionSet, makeItSafe bool) int {
	minLookahead := ts.minLookahead
	maxLookahead := ts.maxLookahead
	spread := maxLookahead - minLookahead + 1
	
	// Make sure we have enough space
	neededSize := at.nAction + spread + at.nsymbol
	if neededSize > len(at.actions) {
		// Grow the table
		newSize := neededSize + len(at.actions)/2 + 20
		newActions := make([]ActionTableEntry, newSize)
		
		// Copy existing entries
		copy(newActions, at.actions)
		
		// Initialize new entries
		for i := len(at.actions); i < newSize; i++ {
			newActions[i].Lookahead = NOT_USED
			newActions[i].Action = NOT_USED
		}
		
		at.actions = newActions
	}
	
	// Find the best offset for this transaction set
	bounds := 0
	if makeItSafe {
		// For terminal symbols, we're more cautious about offset selection
		bounds = minLookahead
	}
	
	// First try to find an exact match in the existing table
	found := false
	bestOffset := 0
	
	// Check each possible offset to see if this transaction already exists
	for i := at.nAction - 1; i >= bounds; i-- {
		if at.actions[i].Lookahead == minLookahead {
			// This could be a match - check all entries
			match := true
			
			for j := 0; j < len(ts.lookaheads); j++ {
				lookahead := ts.lookaheads[j]
				action := ts.actions[j]
				
				// Compute offset into action table
				k := lookahead - minLookahead + i
				
				// Check if this slot exists and matches
				if k < 0 || k >= at.nAction || 
				   at.actions[k].Lookahead != lookahead || 
				   at.actions[k].Action != action {
					match = false
					break
				}
			}
			
			if match {
				found = true
				bestOffset = i - minLookahead
				break
			}
		}
	}
	
	// If no match found, look for an empty slot
	if !found {
		for i := bounds; i < len(at.actions) - (maxLookahead - minLookahead); i++ {
			// Check if this position has enough empty slots
			hasRoom := true
			
			for j := 0; j < len(ts.lookaheads); j++ {
				lookahead := ts.lookaheads[j]
				k := lookahead - minLookahead + i
				
				// Make sure this slot is unused
				if k < 0 || at.actions[k].Lookahead != NOT_USED {
					hasRoom = false
					break
				}
			}
			
			// Also make sure no existing entry would conflict
			if hasRoom {
				for j := 0; j < at.nAction; j++ {
					if at.actions[j].Lookahead >= 0 {
						// Check if this lookahead would map to one of our slots
						if at.actions[j].Lookahead == j + minLookahead - i {
							hasRoom = false
							break
						}
					}
				}
			}
			
			if hasRoom {
				bestOffset = i - minLookahead
				break
			}
		}
		
		// If we get here and bestOffset is still 0, we'll append to the end
		if bestOffset == 0 && !found {
			bestOffset = at.nAction - minLookahead
		}
	}
	
	// Insert the transaction set at the chosen offset
	for j := 0; j < len(ts.lookaheads); j++ {
		lookahead := ts.lookaheads[j]
		action := ts.actions[j]
		
		// If this action equals the default action, we can skip it
		if action == ts.defaultAction {
			continue
		}
		
		// Compute position in the table
		k := lookahead - minLookahead + bestOffset + minLookahead
		
		// Insert the action
		at.actions[k].Lookahead = lookahead
		at.actions[k].Action = action
		
		// Update the table size if needed
		if k >= at.nAction {
			at.nAction = k + 1
		}
	}
	
	// If we're making it safe for terminals, ensure we have enough space
	if makeItSafe && bestOffset + minLookahead + at.nterminal >= at.nAction {
		at.nAction = bestOffset + minLookahead + at.nterminal + 1
	}
	
	return bestOffset + minLookahead
}

// Size returns the actual size of the action table (ignoring trailing empty entries)
func (at *ActionTable) Size() int {
	n := at.nAction
	
	// Trim unused trailing entries
	for n > 0 && at.actions[n-1].Lookahead < 0 {
		n--
	}
	
	return n
}

// compressActionTable takes the State set and generates compressed action and goto tables
func (p *Parser) compressActionTable() ([]ActionTableEntry, []int, []int) {
	// Create the action table
	at := NewActionTable(p.Nsymbol, p.countTerminals())
	
	// Fill in the action offsets for each state
	actionOffsets := make([]int, p.StateSet.NState)
	gotoOffsets := make([]int, p.StateSet.NState)
	
	// Process each state
	for _, state := range p.StateSet.States {
		// First, process terminal symbol actions (shift, reduce, accept, error)
		terminalTrans := CreateTransaction()
		
		for _, action := range state.Actions {
			// Skip actions for non-terminals
			if !action.Sp.IsTerminal {
				continue
			}
			
			// Encode the action
			var actionCode int
			switch action.Type {
			case SHIFT:
				actionCode = action.X // The state to shift to
			case REDUCE:
				actionCode = -action.X - 1 // The rule to reduce by 
			case ACCEPT:
				actionCode = -9999 // Special code for accept
			default:
				actionCode = -1 // Error
			}
			
			// Add to the transaction set
			terminalTrans.AddAction(action.Sp.Index, actionCode)
		}
		
		// Find a default action if possible
		terminalTrans.FindDefaultAction()
		
		// Insert into the action table (makeItSafe=true for terminals)
		actionOffsets[state.StateNum] = at.Insert(terminalTrans, true)
		
		// Now process non-terminal actions (goto)
		nonterminalTrans := CreateTransaction()
		
		for sym, gotoState := range state.Goto {
			// Skip terminal symbols
			if sym.IsTerminal {
				continue
			}
			
			// Add goto action
			nonterminalTrans.AddAction(sym.Index, gotoState.StateNum)
		}
		
		// Find a default goto action if possible
		nonterminalTrans.FindDefaultAction()
		
		// Insert into the goto table (makeItSafe=false for non-terminals)
		gotoOffsets[state.StateNum] = at.Insert(nonterminalTrans, false)
	}
	
	// Build the final arrays
	actSize := at.Size()
	actionsArr := make([]ActionTableEntry, actSize)
	
	// Copy the action table
	copy(actionsArr, at.actions[:actSize])
	
	// Determine the default actions for each state
	defaultActions := make([]int, p.StateSet.NState)
	for i := range defaultActions {
		defaultActions[i] = -1 // No default action initially
	}
	
	// Find states that can use default reductions
	for _, state := range p.StateSet.States {
		bestReduceCount := 0
		bestReduceRule := -1
		
		// Count frequency of reduce actions
		reduceCounts := make(map[int]int)
		
		for _, action := range state.Actions {
			if action.Type == REDUCE {
				reduceCounts[action.X]++
				
				if reduceCounts[action.X] > bestReduceCount {
					bestReduceCount = reduceCounts[action.X]
					bestReduceRule = action.X
				}
			}
		}
		
		// If a rule is used frequently, make it the default
		if bestReduceCount > 1 {
			defaultActions[state.StateNum] = -bestReduceRule - 1
		}
	}
	
	return actionsArr, actionOffsets, gotoOffsets
}