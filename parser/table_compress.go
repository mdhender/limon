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

// CreateTransaction creates a new transaction set
func CreateTransaction() *transactionSet {
	return &transactionSet{
		lookaheads:    make([]int, 0),
		actions:       make([]int, 0),
		minLookahead:  99999,  // Initialize to high value
		maxLookahead: -99999, // Initialize to low value
		defaultAction: -1,     // Default is error action
	}
}

// AddAction adds an action to the transaction set
func (ts *transactionSet) AddAction(lookahead, action int) {
	// Check if lookahead already exists
	for i, la := range ts.lookaheads {
		if la == lookahead {
			// Replace the action if it already exists
			ts.actions[i] = action
			return
		}
	}
	
	// Add the new lookahead and action
	ts.lookaheads = append(ts.lookaheads, lookahead)
	ts.actions = append(ts.actions, action)
	
	// Update min/max
	if lookahead < ts.minLookahead {
		ts.minLookahead = lookahead
	}
	if lookahead > ts.maxLookahead {
		ts.maxLookahead = lookahead
	}
}

// FindDefaultAction determines a default action for this transaction set
func (ts *transactionSet) FindDefaultAction() {
	// Initialize default action to error
	ts.defaultAction = -1
	
	// Count the number of times each action occurs
	actCounts := make(map[int]int)
	for _, action := range ts.actions {
		actCounts[action]++
	}
	
	// Find the most frequent action
	maxCount := 0
	maxAction := -1
	for action, count := range actCounts {
		if count > maxCount {
			maxCount = count
			maxAction = action
		}
	}
	
	// Set the default action to the most frequent action
	if maxCount > 1 { // Only use default if it saves us at least one entry
		ts.defaultAction = maxAction
	}
}

// Size returns the number of entries in the transaction set
func (ts *transactionSet) Size() int {
	return len(ts.lookaheads)
}

// GetAction returns the action for a given lookahead
func (ts *transactionSet) GetAction(lookahead int) int {
	// Find the lookahead
	for i, la := range ts.lookaheads {
		if la == lookahead {
			return ts.actions[i]
		}
	}
	
	// Return the default action if lookahead not found
	return ts.defaultAction
}

// RemoveAction removes an action from the transaction set
func (ts *transactionSet) RemoveAction(lookahead int) {
	// Find the lookahead
	for i, la := range ts.lookaheads {
		if la == lookahead {
			// Remove it
			ts.lookaheads = append(ts.lookaheads[:i], ts.lookaheads[i+1:]...)
			ts.actions = append(ts.actions[:i], ts.actions[i+1:]...)
			
			// Update min/max (may need to recalculate)
			if lookahead == ts.minLookahead || lookahead == ts.maxLookahead {
				ts.minLookahead = 99999
				ts.maxLookahead = -99999
				for _, la := range ts.lookaheads {
					if la < ts.minLookahead {
						ts.minLookahead = la
					}
					if la > ts.maxLookahead {
						ts.maxLookahead = la
					}
				}
			}
			break
		}
	}
}

// GetIsSparse returns true if the action table is sparse
func (ts *transactionSet) GetIsSparse() bool {
	// If there are no actions, it's not sparse
	if len(ts.lookaheads) == 0 {
		return false
	}
	
	// Calculate density
	range_size := ts.maxLookahead - ts.minLookahead + 1
	density := float64(len(ts.lookaheads)) / float64(range_size)
	
	// If density is less than 0.5, it's considered sparse
	return density < 0.5
}

// Insert adds a transaction set to the action table
func (at *ActionTable) Insert(ts *transactionSet, makeItSafe bool) int {
	// If no actions to insert, return 0
	if ts.Size() == 0 {
		return 0
	}
	
	// Optimize: pre-compute non-default actions
	nonDefaults := make(map[int]int) // map[lookahead]action
	for i, la := range ts.lookaheads {
		// Only keep actions that are different from the default
		if ts.actions[i] != ts.defaultAction {
			nonDefaults[la] = ts.actions[i]
		}
	}
	
	// Find the best place to insert the transaction set
	best_offset := 0
	best_collisions := 9999
	
	// Try different offsets to find the best fit
	for offset := 0; offset < at.nAction+len(nonDefaults)+10; offset++ {
		collisions := 0
		
		// Check each non-default action
		for la := range nonDefaults {
			// Calculate the index in the action table
			table_index := offset + la
			
			// Ensure we have enough space in the action table
			if table_index >= len(at.actions) {
				// Resize the action table
				new_size := table_index + 100
				new_actions := make([]ActionTableEntry, new_size)
				copy(new_actions, at.actions)
				
				// Initialize new entries
				for i := len(at.actions); i < new_size; i++ {
					new_actions[i].Lookahead = NOT_USED
					new_actions[i].Action = NOT_USED
				}
				
				at.actions = new_actions
			}
			
			// Check if there's a collision at this offset
			if at.actions[table_index].Lookahead != NOT_USED {
				collisions++
			}
		}
		
		// If this offset has fewer collisions, it's the new best
		if collisions < best_collisions {
			best_collisions = collisions
			best_offset = offset
			
			if collisions == 0 {
				break // Perfect fit, no need to search further
			}
		}
	}
	
	// Now insert the transaction set at the best offset
	for la, action := range nonDefaults {
		table_index := best_offset + la
		
		// Ensure we have enough space in the action table
		if table_index >= len(at.actions) {
			// Resize the action table
			new_size := table_index + 100
			new_actions := make([]ActionTableEntry, new_size)
			copy(new_actions, at.actions)
			
			// Initialize new entries
			for i := len(at.actions); i < new_size; i++ {
				new_actions[i].Lookahead = NOT_USED
				new_actions[i].Action = NOT_USED
			}
			
			at.actions = new_actions
		}
		
		// Insert the entry
		at.actions[table_index].Lookahead = la
		at.actions[table_index].Action = action
		
		// Update nAction to reflect the highest used index
		if table_index >= at.nAction {
			at.nAction = table_index + 1
		}
	}
	
	// Return the offset for this transaction set
	return best_offset
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
			if action.Sp != nil && !action.Sp.IsTerminal {
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
			if action.Sp == nil {
				// End of input symbol (for ACCEPT actions)
				terminalTrans.AddAction(0, actionCode)
			} else {
				terminalTrans.AddAction(action.Sp.Index, actionCode)
			}
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
	
	return at.actions[:at.nAction], actionOffsets, gotoOffsets
}