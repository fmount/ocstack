module github.com/fmount/ocstack

go 1.24.0

toolchain go1.24.2

require github.com/ollama/ollama v0.6.8

// Replace llm module as we work with a single repo
replace github.com/fmount/ocstack/gollm => ./gollm

// Replace templates module as we work with a single repo
replace github.com/fmount/ocstack/template => ./template

// Replace tools module as we work with a single repo
replace github.com/fmount/ocstack/tools => ./tools
