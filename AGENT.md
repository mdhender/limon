# Agent Guidelines for Lemon Parser Generator

## Build Commands
- Build and run Go implementation: `go run main.go grammar_file.y`
- Build lemon binary: `go build -o limon`
- Run tests: `go test ./...`
- Build original C version: `gcc tool/lemon.c -o lemon` or `cc -o lemon tool/lemon.c`
- Run original C version: `./lemon grammar_file.y`

## Code Style Guidelines
- Go code follows standard Go conventions and idioms
- Error handling: Return errors rather than using panics
- Use Go's standard library packages when possible
- Terminal symbols start with uppercase letters (typically ALL_CAPS) 
- Non-terminal symbols start with lowercase letters (typically all_lowercase)
- Prefer left recursion over right recursion for better parser performance
- Use left-associative operators when possible to reduce stack size

## Project Structure
- `parser/` - Go implementation of the parser generator
  - `parser.go` - Main parser implementation 
  - `grammar.go` - Grammar data structures
- `tool/` - Original C implementation
  - `lemon.c` - Original parser generator source code
  - `lempar.c` - Original parser template file
- `doc/` - Documentation files
- `main.go` - Command-line interface