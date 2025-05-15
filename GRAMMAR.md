# Lemon Grammar File Syntax

This document describes the grammar of the Lemon Parser Generator input files. The grammar is written in Lemon grammar notation.

## Grammar

```lemon
/* Top-level grammar file structure */
grammar ::= declaration_list.

/* Multiple declarations */
declaration_list ::= declaration_list declaration.
declaration_list ::= declaration.

/* Individual declaration types */
declaration ::= rule.
declaration ::= directive.

/* Grammar rule definition */
rule ::= symbol COLON_COLON_EQUAL symbol_list period action_opt precedence_opt.
rule ::= symbol COLON_COLON_EQUAL period action_opt precedence_opt.

/* Symbol list for the right-hand side of a rule */
symbol_list ::= symbol_list symbol.
symbol_list ::= symbol.

/* A single grammar symbol with optional alias */
symbol ::= SYMBOL.
symbol ::= SYMBOL PAREN_LEFT SYMBOL PAREN_RIGHT.

/* Optional code action block after a rule */
action_opt ::= ACTION_BLOCK.
action_opt ::= .

/* Optional precedence marker after a rule */
precedence_opt ::= BRACKET_LEFT SYMBOL BRACKET_RIGHT.
precedence_opt ::= .

/* Period at the end of a rule */
period ::= DOT.

/* Different types of directives */
directive ::= code_directive.
directive ::= default_destructor_directive.
directive ::= default_type_directive.
directive ::= destructor_directive.
directive ::= extra_argument_directive.
directive ::= fallback_directive.
directive ::= include_directive.
directive ::= name_directive.
directive ::= parse_accept_directive.
directive ::= parse_failure_directive.
directive ::= precedence_directive.
directive ::= stack_overflow_directive.
directive ::= stack_size_directive.
directive ::= start_symbol_directive.
directive ::= syntax_error_directive.
directive ::= token_directive.
directive ::= token_destructor_directive.
directive ::= token_prefix_directive.
directive ::= token_type_directive.
directive ::= type_directive.
directive ::= wildcard_directive.

/* List of directives */
code_directive               ::= PCT_CODE               CODE_BLOCK.
default_destructor_directive ::= PCT_DEFAULT_DESTRUCTOR CODE_BLOCK.
default_type_directive       ::= PCT_DEFAULT_TYPE       CODE_BLOCK.
destructor_directive         ::= PCT_DESTRUCTOR SYMBOL  CODE_BLOCK.
extra_argument_directive     ::= PCT_EXTRA_ARGUMENT     CODE_BLOCK.
include_directive            ::= PCT_INCLUDE            CODE_BLOCK.
parse_accept_directive       ::= PCT_PARSE_ACCEPT       CODE_BLOCK.
parse_failure_directive      ::= PCT_PARSE_FAILURE      CODE_BLOCK.
stack_overflow_directive     ::= PCT_STACK_OVERFLOW     CODE_BLOCK.
syntax_error_directive       ::= PCT_SYNTAX_ERROR       CODE_BLOCK.
token_destructor_directive   ::= PCT_TOKEN_DESTRUCTOR   CODE_BLOCK.
token_type_directive         ::= PCT_TOKEN_TYPE         CODE_BLOCK.
type_directive               ::= PCT_TYPE SYMBOL        CODE_BLOCK.

stack_size_directive         ::= PCT_STACK_SIZE         INTEGER.

name_directive               ::= PCT_NAME               SYMBOL.
start_symbol_directive       ::= PCT_START_SYMBOL       SYMBOL.
token_prefix_directive       ::= PCT_TOKEN_PREFIX       SYMBOL.
wildcard_directive           ::= PCT_WILDCARD           SYMBOL DOT.

fallback_directive           ::= PCT_FALLBACK           terminal_list DOT.
precedence_directive         ::= PCT_LEFT               terminal_list DOT.
precedence_directive         ::= PCT_NONASSOC           terminal_list DOT.
precedence_directive         ::= PCT_RIGHT              terminal_list DOT.
token_directive              ::= PCT_TOKEN              terminal_list DOT.

/* List of terminal symbols for directives */
terminal_list ::= terminal_list TERMINAL.
terminal_list ::= TERMINAL.
```

## Token Definitions

The grammar assumes the following tokens are provided by the lexer:

- `ACTION_BLOCK`: C code enclosed in braces { ... } for action blocks after rules
- `BRACKET_LEFT`, `BRACKET_RIGHT`: Square brackets used for precedence overrides
- `CODE_BLOCK`: C code enclosed in braces { ... }
- `DOT`: A period character (.)
- `INTEGER`: A numeric literal
- `PAREN_LEFT`, `PAREN_RIGHT`: Parentheses used for symbol aliases
- `COLON_COLON_EQUAL`: The ::= token that separates LHS from RHS in rules
- `SYMBOL`: A grammar symbol (terminal or non-terminal)
- `TERMINAL`: A terminal symbol (starts with uppercase letter)

## Directive Tokens

The grammar assumes that the lexer provides tokens for the following directives:

- `PCT_CODE`: The %code directive
- `PCT_DEFAULT_DESTRUCTOR`: The %default_destructor directive
- `PCT_DEFAULT_TYPE`: The %default_type directive
- `PCT_DESTRUCTOR`: The %destructor directive
- `PCT_EXTRA_ARGUMENT`: The %extra_argument directive
- `PCT_FALLBACK`: The %fallback directive
- `PCT_INCLUDE`: The %include directive
- `PCT_LEFT`: The %left directive
- `PCT_NAME`: The %name directive
- `PCT_NONASSOC`: The %nonassoc directive
- `PCT_PARSE_ACCEPT`: The %parse_accept directive
- `PCT_PARSE_FAILURE`: The %parse_failure directive
- `PCT_RIGHT`: The %right directive
- `PCT_STACK_OVERFLOW`: The %stack_overflow directive
- `PCT_STACK_SIZE`: The %stack_size directive
- `PCT_START_SYMBOL`: The %start_symbol directive
- `PCT_SYNTAX_ERROR`: The %syntax_error directive
- `PCT_TOKEN`: The %token directive
- `PCT_TOKEN_DESTRUCTOR`: The %token_destructor directive
- `PCT_TOKEN_PREFIX`: The %token_prefix directive
- `PCT_TOKEN_TYPE`: The %token_type directive
- `PCT_TYPE`: The %type directive
- `PCT_WILDCARD`: The %wildcard directive

## Notes

1. Terminal symbols start with uppercase letters (e.g., TOKEN, ID, NUMBER)
2. Non-terminal symbols start with lowercase letters (e.g., expr, stmt, program)
3. The ::= symbol separates the left-hand side from the right-hand side of a rule
4. Each rule must end with a period (.)
5. Actions (C code blocks) can follow immediately after the period
6. Precedence overrides can be specified in square brackets after the period
7. Symbol aliases can be specified in parentheses after a symbol name
8. Empty right-hand sides (epsilon productions) are specified by omitting symbols on the right-hand side
9. Comments (C or C++ style) are accepted and silently passed through to the code generator