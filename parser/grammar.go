package parser

// Symbol represents a terminal or non-terminal symbol in the grammar
type Symbol struct {
	Name       string  // Name of the symbol
	Index      int     // Index number for this symbol
	Type       string  // Declared type of this symbol
	Rule       *Rule   // Linked list of rules that use this symbol
	Fallback   *Symbol // Fallback token in case this token doesn't parse
	Prec       int     // Precedence if defined (-1 otherwise)
	Assoc      int     // Associativity if precedence is defined
	FirstSet   []*Symbol // First-set for all rules of this symbol
	Lambda     bool    // True if NT can generate an empty string
	Destructor string  // Code which executes whenever this symbol is popped
	Dtnum      int     // Number which determines destructor action
	IsTerminal bool    // True for terminal symbols, false for non-terminals
}

// Rule represents a grammar rule
type Rule struct {
	Lhs        *Symbol   // Left-hand side of the rule
	LhsAlias   string    // Alias for the LHS (NULL if none)
	Rhs        []*Symbol // Right-hand side symbols
	RhsAlias   []string  // Alias for each RHS symbol (NULL if none)
	LineNo     int       // Line number for the rule
	RuleNum    int       // Rule number for this rule
	CodeUsed   bool      // True if the action contains variables
	NoCode     bool      // True if this rule has no associated code
	Precedence *Symbol   // Precedence symbol for this rule
	Index      int       // An index number used to distinguish rules
	CanReduce  bool      // True if this rule is ever reduced
	NextLhs    *Rule     // Next rule with the same LHS
	Next       *Rule     // Next rule in the global list
}

// State represents a state in the finite state machine
type State struct {
	Bp         []int     // The basis configurations for this state
	Configs    []int     // All configurations in this state
	StateNum   int       // Sequential number for this state
	Action     []Action  // Array of actions for this state
	NTAction   []Action  // Array of actions on non-terminals
	Goto       []int     // Goto destinations for each non-terminal
	Next       *State    // Linked list of states
}

// Action is something that occurs when a token is encountered
type Action struct {
	Sp         *Symbol   // The look-ahead symbol
	Type       int       // The type of action - shift, reduce, accept, error
	X          int       // Extra information about the action
	Next       *Action   // Next action in the list
}

// These values for the Action.type field represent the various actions
const (
	SHIFT  = iota
	REDUCE
	ACCEPT
	ERROR
)

// Precedence values for symbols
const (
	NONE       = iota
	LEFT
	RIGHT
	NONASSOC
)