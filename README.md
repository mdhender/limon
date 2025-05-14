# Lemon Parser Generator - Go Implementation

## Overview

Limon is a Go implementation of the Lemon LALR(1) parser generator, originally developed for SQLite. This port maintains full compatibility with the original C implementation while taking advantage of Go's features.

## Features

- Generates reentrant, thread-safe parsers
- No global variables
- Built-in memory management with destructors
- Robust error handling
- Support for special features:
  - Epsilon productions
  - Fallback tokens
  - Wildcard tokens

## Building

```bash
go build -o limon
```

## Basic Usage

```bash
./limon grammar.y
```

This generates three files:
- `grammar.c` - The parser implementation
- `grammar.h` - Header file with token definitions
- `grammar.out` - Information about the parser state machine

## Documentation

Detailed documentation can be found in the `doc/` directory:

- `doc/lemon.md` - Original Lemon documentation
- `doc/go-lemon.md` - Go-specific documentation and extensions

## Examples

Check the `examples/` directory for sample grammar files showing various Lemon features.

## License

This project is in the public domain, following the original Lemon's licensing.