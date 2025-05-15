// Sample calculator grammar demonstrating Lemon special features
// This grammar shows epsilon productions, fallback tokens, and wildcard tokens

%token_type    {float64}   // Define the type of all terminals
%default_type  {float64}   // Define the default type for non-terminals

%left PLUS MINUS.
%left TIMES DIVIDE.
%right POW.

// Wildcard token to handle any unknown token as an error
%wildcard ERROR.

// Fallback tokens: keywords can fall back to being identifiers
%fallback ID VAR PRINT IF WHILE.

// The main program is a list of statements
program ::= stmt_list.

// A statement list can be empty (epsilon production)
stmt_list ::= stmt_list stmt.
stmt_list ::= .

// Statement types
stmt ::= expr SEMI.          { printf("Result: %f\n", $expr); }
stmt ::= VAR ID ASSIGN expr SEMI.  { setVariable($ID, $expr); }
stmt ::= PRINT expr SEMI.    { printf("%f\n", $expr); }

expr(A) ::= expr(B) PLUS expr(C).   { A = B + C; }
expr(A) ::= expr(B) MINUS expr(C).  { A = B - C; }
expr(A) ::= expr(B) TIMES expr(C).  { A = B * C; }
expr(A) ::= expr(B) DIVIDE expr(C). { A = B / C; }
expr(A) ::= expr(B) POW expr(C).    { A = pow(B, C); }
expr(A) ::= LPAREN expr(B) RPAREN.  { A = B; }
expr(A) ::= NUMBER(B).              { A = B; }
expr(A) ::= ID(B).                 { A = getVariable(B); }

// Any unrecognized input token will match this rule due to the wildcard
stmt ::= ERROR. { printf("Syntax error, unexpected token\n"); }

%code {
// These functions would be implemented in the actual application
// They're just placeholders here for the example
float64 getVariable(char* name) {
  // Implementation to retrieve variable value
  return 0.0;
}

void setVariable(char* name, float64 value) {
  // Implementation to set variable value
}
}