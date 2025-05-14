# Go Lemon Parser Generator

## Overview

Go Lemon is a port of the Lemon LALR(1) parser generator from C to Go. It maintains full compatibility with the original Lemon tool while taking advantage of Go's modern features.

Lemon creates parsers that are:

- **Reentrant and thread-safe** - Unlike yacc/bison parsers, Lemon parsers have no global variables
- **Clean API design** - The tokenizer calls the parser, not the other way around
- **Memory leak resistant** - Built-in destructor support for automatic cleanup
- **Robust error handling** - Sophisticated error recovery mechanisms

## Quick Start

### Building

```bash
go build -o limon
```

### Basic Usage

```bash
./limon grammar.y
```

This generates three files:
- `grammar.c` - The parser implementation
- `grammar.h` - Header file with token definitions
- `grammar.out` - Information about the parser state machine

## Command Line Options

| Option | Description |
|--------|-------------|
| `-b` | Show only the basis for each parser state in the report file |
| `-c` | Do not compress action tables (helps with earlier error detection) |
| `-d <directory>` | Write output files to specified directory |
| `-g` | Print grammar without actions instead of generating parser |
| `-l` | Omit #line directives in output |
| `-m` | Generate makeheaders-compatible output |
| `-p` | Display all precedence-resolved conflicts |
| `-q` | Suppress generation of the report file |
| `-r` | Do not sort or renumber states |
| `-s` | Show parser statistics |
| `-T <file>` | Use specified template file instead of default |

## Special Features

### Epsilon Productions

Lemon supports rules with empty right-hand sides, known as epsilon productions:

```
optional_clause ::= clause.
optional_clause ::= . // Epsilon production
```

During parsing, epsilon productions are handled automatically, generating appropriate transitions in the state machine and setting the `Lambda` flag on non-terminals that can derive the empty string.

### Fallback Tokens

Fallback tokens allow a token to be treated as an alternative token when a syntax error would otherwise occur.

```
%fallback ID WHILE FOR IF .
```

In this example, if WHILE, FOR, or IF would cause a syntax error, the parser will try treating them as ID tokens instead. This is particularly useful for handling languages with many keywords that could also be identifiers.

### Wildcard Tokens

Wildcard tokens match any input token when no other match is possible:

```
%wildcard ANY .
stmt ::= ANY . { /* Handle any token as a statement */ }
```

Wildcard tokens provide a flexible way to create catch-all rules in your grammar.

## Template Customization

If you create your own parser template (instead of using the default lempar.c), make sure to preserve these special features:

1. For **fallback tokens**, include the YYFALLBACK conditional blocks
2. For **wildcard tokens**, include the YYWILDCARD conditional blocks
3. For **epsilon productions**, ensure the parser can handle empty right-hand sides

These features are implemented by conditional code sections in the template that must be preserved for full functionality.

## Parser Interface

Lemon generates a parser with this interface (assuming default function names):

```c
void *ParseAlloc(void *(*mallocProc)(size_t));
void ParseFree(void *p, void (*freeProc)(void*));
void Parse(void *yyp, int yymajor, ParseTOKENTYPE yyminor);
void ParseTrace(FILE *TraceFILE, char *zTracePrompt);
```

See the official Lemon documentation for details on using these functions.

## References

- Original Lemon documentation: https://sqlite.org/src/doc/trunk/doc/lemon.html
- SQLite project (original home of Lemon): https://sqlite.org