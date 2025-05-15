/* Simple calculator grammar */
%token_type {int}
%token INTEGER.
%left PLUS MINUS.
%left TIMES DIVIDE.

%syntax_error {printf("Syntax error!\n");}

%include {
#include <stdio.h>
#include <stdlib.h>
}

program ::= expr(A). { printf("Result: %d\n", A); }

expr(A) ::= expr(B) PLUS expr(C).   { A = B + C; }
expr(A) ::= expr(B) MINUS expr(C).  { A = B - C; }
expr(A) ::= expr(B) TIMES expr(C).  { A = B * C; }
expr(A) ::= expr(B) DIVIDE expr(C). { A = B / C; }
expr(A) ::= INTEGER(B).             { A = B; }