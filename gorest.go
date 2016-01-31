package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"

	"github.com/jsaund/gorest/generate"
	"github.com/jsaund/gorest/parse"
)

var (
	input  = flag.String("input", "", "name of input file containing REST API to generate (if absent then Stdin is used)")
	output = flag.String("output", "", "name of output file containing generated API request and response implementation")
	pkg    = flag.String("pkg", "", "name of output file package (should be the same as input package)")
)

func main() {
	flag.Parse()

	if *output == "" {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "Expects valid output filename")
		os.Exit(1)
	}

	if *pkg == "" {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "Expects valid package name")
		os.Exit(1)
	}

	var file *ast.File
	fileset := token.NewFileSet()

	if *input != "" {
		f, err := parser.ParseFile(fileset, *input, nil, parser.ParseComments)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse input filename. Is input filename %s valid?\n", *input)
			os.Exit(1)
		}
		file = f
	} else {
		f, err := parser.ParseFile(fileset, "", os.Stdin, parser.ParseComments)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to parse input file. Is source input valid?")
			os.Exit(1)
		}
		file = f
	}

	parseResult := parseAST(file, *pkg)
	buf, err := generateBuilder(parseResult)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate REST API implementation. %s\n", err)
		os.Exit(1)
	}

	if err := writeFile(*output, buf); err != nil {
		log.Fatalf("Failed to write generated source to file %s. Reason: %s", *output, err)
	}

	fmt.Println("Generated source written to file " + *output)
}

// parseAST walks the AST represented by the interface we wish to generate an implementation for.
// Returns ParseResult which contains request and response implementation details.
func parseAST(file *ast.File, pkg string) *parse.ParseResult {
	parser := parse.NewParser(file, pkg)
	return parser.Parse()
}

// generateBuilder transforms the parsed information in to a request builder and response golang file.
func generateBuilder(r *parse.ParseResult) ([]byte, error) {
	return generate.Generate(r)
}

// writeFile persists the data to the specified file
func writeFile(filename string, data []byte) error {
	return ioutil.WriteFile(filename, data, 0644)
}
