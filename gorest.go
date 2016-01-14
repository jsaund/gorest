package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"text/template"
)

var (
	input   = flag.String("input", "", "name of input file containing REST API to generate (if absent then Stdin is used)")
	output  = flag.String("output", "", "name of output file containing generated API request and response implementation")
	pkg     = flag.String("pkg", "", "name of output file package (should be the same as input package)")
	funcMap = template.FuncMap{
		"ParamsList": getParamsList,
		"ParamName":  getParamName,
		"ParamKey":   getParamKey,
		"MethodName": getMethodName,
	}
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
			fmt.Fprintln(os.Stderr, "Failed to parse input filename. Is input filename %s valid?", *input)
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

	buf, err := generate(file, *pkg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to generate REST API implementation. %s", err)
		os.Exit(1)
	}

	if err := ioutil.WriteFile(*output, buf, 0644); err != nil {
		log.Fatalf("Failed to write generated source to file %s. Reason: %s", *output, err)
	}

	fmt.Println("Generated source written to file %s", *output)
}

func generate(file *ast.File, pkg string) ([]byte, error) {
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	visitor := newVisitor(info, pkg)
	ast.Walk(visitor, file)

	return generateRequestResponse(visitor.generateInfo)
}

func generateRequestResponse(r *generateInfo) ([]byte, error) {
	var builderTemplate = template.Must(template.New("builder").Funcs(funcMap).Parse(`/*
* CODE GENERATED AUTOMATICALLY WITH Go500px
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/

package {{.Pkg}}

import (
	"net/http"
	"net/url"
)

{{ if .CallbackType }}
type {{ $.CallbackType }} interface {
	OnStart()
	OnError(reason string)
	OnSuccess(response {{ $.ResponseType }})
}
{{ end }}

type {{ .RequestType }}Impl struct {
	baseUrl     string
	queryParams url.Values
}

func New{{ .RequestType }}(baseUrl string) {{ .RequestType }} {
	return &{{ .RequestType }}Impl{
		baseUrl: baseUrl,
		queryParams: url.Values{},
	}
}

{{ range $key, $value := .QueryParams }}
func (b *{{ $.RequestType }}Impl) {{ $key }}({{ ParamsList $value.Type }}) {{ $.RequestType }} {
	b.queryParams.Add("{{ $value | ParamKey }}", {{ ParamName $value.Type true 0 }})
	return b
}
{{ end }}

func (b *{{ .RequestType }}Impl) build() (*http.Request, error) {
	req, err := http.NewRequest("{{ .HttpMethod }}", b.baseUrl + "{{ .ApiEndpoint }}", nil)
	if err != nil {
		return nil, err
	}
	if len(b.queryParams) > 0 {
		req.URL.RawQuery = b.queryParams.Encode()
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}

{{ if and .ResponseType .SyncResponse }}
func (b *{{ $.RequestType }}Impl) {{ $.SyncResponse | MethodName }}() ({{ $.ResponseType }}, error) {
	request, err := b.build()
	if err != nil {
		return nil, err
	}
	request.URL.RawQuery = request.URL.Query().Encode()

	response, err := getClient().Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	result, err := New{{ $.ResponseType }}(response.Body)
	if err != nil {
		return nil, err
	}
	return result, nil
}
{{ end }}

{{ if and .CallbackType .AsyncResponse }}
func (b *{{ $.RequestType }}Impl) {{ $.AsyncResponse | MethodName }}({{ ParamsList $.AsyncResponse.Type }}) {
	if {{ ParamName $.AsyncResponse.Type false 0 }} != nil {
		{{ ParamName $.AsyncResponse.Type false 0 }}.OnStart()
	}

	go func(b *{{ $.RequestType }}Impl) {
		response, err := b.{{ $.SyncResponse | MethodName }}()

		if {{ ParamName $.AsyncResponse.Type false 0 }} != nil {
			if err != nil {
				{{ ParamName $.AsyncResponse.Type false 0 }}.OnError(err.Error())
			} else {
				{{ ParamName $.AsyncResponse.Type false 0 }}.OnSuccess(response)
			}
		}
	}(b)
}
{{ end }}
`))
	var buf bytes.Buffer
	err := builderTemplate.Execute(&buf, r)
	if err != nil {
		log.Fatalf("Failed to generate template: %v", err)
		return nil, err
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("Failed to generate template: %v", err)
		return nil, err
	}

	return formatted, nil
}

// Return true if os.Stdin appears to be interactive
func isInteractive() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fileInfo.Mode()&(os.ModeCharDevice|os.ModeCharDevice) != 0
}

func getMethodName(f *ast.Field) string {
	return f.Names[0].Name
}

func getParamKey(f *ast.Field) string {
	re := regexp.MustCompile(pattern)
	comment := f.Doc.Text()
	key := extractQueryParam(re, comment)
	if key == "" {
		log.Fatalf("Must have query parameter defined.")
	}
	return key
}

func getParamName(e ast.Expr, forceString bool, index int) string {
	function, ok := e.(*ast.FuncType)
	if !ok {
		log.Fatalf("Expression must be function type")
		return ""
	}
	p := function.Params
	if len(p.List) == 0 {
		log.Fatalf("Function does not have any parameters")
		return ""
	}

	if index >= len(p.List) {
		log.Fatalf("Illegal parameter index %d. Number of parameters for function is %d", index, len(p.List))
		return ""
	}

	paramName := p.List[index].Names[0].Name
	if forceString {
		paramName = fmt.Sprintf("string(%s)", paramName)
	}
	return paramName
}

func getParamsList(e ast.Expr) string {
	p := e.(*ast.FuncType).Params
	var s string
	for i := 0; i < len(p.List); i++ {
		f := p.List[i]
		s += fmt.Sprintf("%s %s", f.Names[0].Name, getParamType(f.Type))
		if i != len(p.List)-1 {
			s += ","
		}
	}
	return s
}

func getParamType(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return "*" + getParamType(v.X)
	case *ast.SelectorExpr:
		return getParamType(v.X) + "." + getParamType(v.Sel)
	default:
		log.Fatalf("Unrecognized expression type: %v", e)
		return ""
	}
}

const (
	sync             string = "SYNC"
	async            string = "ASYNC"
	query            string = "QUERY"
	body             string = "BODY"
	httpMethodGet    string = "GET"
	httpMethodPost   string = "POST"
	httpMethodPut    string = "PUT"
	httpMethodDelete string = "DELETE"
	httpMethodHead   string = "HEAD"

	pattern string = `@(\w+)\(\"(.*)\"\)`
)

type generateInfo struct {
	Pkg           string
	RequestType   string
	ApiEndpoint   string
	HttpMethod    string
	QueryParams   map[string]*ast.Field
	SyncResponse  *ast.Field
	AsyncResponse *ast.Field
	CallbackType  string
	ResponseType  string
}

type astVisitor struct {
	info         *types.Info
	generateInfo *generateInfo
	re           *regexp.Regexp
	buildRequest bool
}

func newVisitor(info *types.Info, pkg string) *astVisitor {
	return &astVisitor{
		info: info,
		generateInfo: &generateInfo{
			Pkg:         pkg,
			QueryParams: make(map[string]*ast.Field),
		},
		re: regexp.MustCompile(pattern),
	}
}

func (v *astVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return v
	}

	switch node.(type) {
	case *ast.File:
		// At the start of a file
		// Reset builder flags
		v.buildRequest = false
		break
	case *ast.TypeSpec:
		// Check if we are at the beginning of a request builder declaration
		// or a response / callback declaration
		// This must be an interface
		typeSpec := node.(*ast.TypeSpec)
		switch typeSpec.Type.(type) {
		case *ast.InterfaceType:
			if v.buildRequest {
				v.generateInfo.RequestType = typeSpec.Name.Name
			} else {
				v.buildRequest = false
			}
			break
		}
		break
	case *ast.InterfaceType:
		if !v.buildRequest {
			// Only interested in parsing methods of the Requeset interface
			// Ignore all other types
			break
		}
		// Retain a mapping of interface methods to their fields which contain
		// the query parameter and argument name and type information to implement
		// the interface
		ifc := node.(*ast.InterfaceType)
		methods := ifc.Methods
		for _, f := range methods.List {
			if qp := extractQueryParam(v.re, f.Doc.List[0].Text); qp != "" {
				v.generateInfo.QueryParams[f.Names[0].Name] = f
			} else if asyncTag := extractAsyncTag(v.re, f.Doc.List[0].Text); asyncTag != "" {
				v.generateInfo.AsyncResponse = f
				v.generateInfo.CallbackType = asyncTag
			} else if syncTag := extractSyncTag(v.re, f.Doc.List[0].Text); syncTag != "" {
				v.generateInfo.SyncResponse = f
				v.generateInfo.ResponseType = syncTag
			}
		}
		break
	case *ast.Comment:
		comment := node.(*ast.Comment)
		if httpMethod, api := extractHttpMethodTag(v.re, comment.Text); httpMethod != "" {
			// Extract the HTTP Method and API from the Interface declaration
			v.buildRequest = true
			v.generateInfo.HttpMethod = httpMethod
			v.generateInfo.ApiEndpoint = api
		}
		break
	}

	return v
}

func extractSyncTag(re *regexp.Regexp, s string) string {
	match := re.FindStringSubmatch(s)
	if len(match) == 3 && isAnnotationSync(match[1]) {
		return match[2]
	}
	return ""
}

func extractAsyncTag(re *regexp.Regexp, s string) string {
	match := re.FindStringSubmatch(s)
	if len(match) == 3 && isAnnotationAsync(match[1]) {
		return match[2]
	}
	return ""
}

func extractHttpMethodTag(re *regexp.Regexp, s string) (string, string) {
	match := re.FindStringSubmatch(s)
	if len(match) == 3 && isAnnotationHttpMethod(match[1]) {
		return match[1], match[2]
	}
	return "", ""
}

func isAnnotationAsync(s string) bool {
	return s == async
}

func isAnnotationSync(s string) bool {
	return s == sync
}

func isAnnotationHttpMethod(s string) bool {
	return s == httpMethodGet || s == httpMethodPost || s == httpMethodPut || s == httpMethodHead || s == httpMethodDelete
}

func extractQueryParam(re *regexp.Regexp, s string) string {
	match := re.FindStringSubmatch(s)
	if len(match) == 3 && match[1] == query {
		return match[2]
	}
	return ""
}
