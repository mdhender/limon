# Agent Guidelines for Lemon Parser Generator

## Build Commands
- Build and run Go implementation: `go run ./... -d generated/ examples/grammar_file.y`
- Build lemon binary: `go build -o build/limon`
- Run tests: `go test ./...`
- Build original C version: `gcc tool/lemon.c -o build/lemon` or `cc -o build/lemon tool/lemon.c`
- Run original C version: `./build/lemon grammar_file.y`

## Code Style Guidelines
- Go code follows standard Go conventions and idioms
- Error handling: Return errors rather than using panics
- Use Go's standard library packages when possible
- Terminal symbols start with uppercase letters (typically ALL_CAPS) 
- Non-terminal symbols start with lowercase letters (typically all_lowercase)
- Prefer left recursion over right recursion for better parser performance
- Use left-associative operators when possible to reduce stack size
- Add `TODO` comments for work that needs to be completed in the future

## Project Structure
- `main.go` - Command-line interface
- `build/` - Path for production deployment files
- `doc/` - Documentation files
- `cli/lemon/` - Hand-written Go implementation of the Lemon parser generator; ignore this path
- `examples/` - Sample Lemon grammars
- `generated/` - Path for temporary and testing files
- `parser/` - Go implementation of the parser generator
  - `parser.go` - Main parser implementation 
  - `grammar.go` - Grammar data structures
- `tool/` - Original C implementation
  - `lemon.c` - Original parser generator source code
  - `lempar.c` - Original parser template file

