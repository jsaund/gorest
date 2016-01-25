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
		"ParamsList":      getParamsList,
		"ParamName":       getParamName,
		"AnnotationValue": getAnnotationValue,
		"FunctionName":    getFunctionName,
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

	info := getInfo(file, *pkg)
	buf, err := generateRequestResponse(info)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate REST API implementation. %s\n", err)
		os.Exit(1)
	}

	if err := ioutil.WriteFile(*output, buf, 0644); err != nil {
		log.Fatalf("Failed to write generated source to file %s. Reason: %s", *output, err)
	}

	fmt.Println("Generated source written to file " + *output)
}

// getInfo walks the AST represented by the interface we wish to generate an implementation for.
// Returns generateInfo which contains request and response implementation details.
func getInfo(file *ast.File, pkg string) *generateInfo {
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	visitor := newVisitor(info, pkg)
	ast.Walk(visitor, file)
	return visitor.generateInfo
}

// generateRequestResponse generates the implementation using the details contained in genereateInfo.
func generateRequestResponse(r *generateInfo) ([]byte, error) {
	var builderTemplate = template.Must(template.New("builder").Funcs(funcMap).Parse(`/*
* CODE GENERATED AUTOMATICALLY WITH GOREST (github.com/jsaund/gorest)
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/

package {{.Pkg}}

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

{{ if .CallbackType }}
type {{ $.CallbackType }} interface {
	OnStart()
	OnError(reason string)
	OnSuccess(response {{ $.ResponseType }})
}
{{ end }}

type {{ .RequestType }}Impl struct {
	baseUrl            string
	pathSubstitutions  map[string]string
	queryParams        url.Values
	postFormParams     url.Values
	postBody           interface{}
	postMultiPartParam map[string][]byte
	headerParams       map[string]string
}

func New{{ .RequestType }}(baseUrl string) {{ .RequestType }} {
	return &{{ .RequestType }}Impl{
		baseUrl:            baseUrl,
		pathSubstitutions:  make(map[string]string),
		queryParams:        url.Values{},
		postFormParams:     url.Values{},
		postMultiPartParam: make(map[string][]byte),
		headerParams:       make(map[string]string),
	}
}

{{ range $key, $value := .PathSubstitutions }}
func (b *{{ $.RequestType }}Impl) {{ $key }}({{ ParamsList $value.Type }}) {{ $.RequestType }} {
	b.pathSubstitutions["{{ AnnotationValue $value }}"] = {{ ParamName $value.Type true 0 }}
	return b
}
{{ end }}

{{ range $key, $value := .QueryParams }}
func (b *{{ $.RequestType }}Impl) {{ $key }}({{ ParamsList $value.Type }}) {{ $.RequestType }} {
	b.queryParams.Add("{{ AnnotationValue $value }}", {{ ParamName $value.Type true 0 }})
	return b
}
{{ end }}

{{ range $key, $value := .PostFormParams }}
func (b *{{ $.RequestType }}Impl) {{ $key }}({{ ParamsList $value.Type }}) {{ $.RequestType }} {
	b.postFormParams.Add("{{ AnnotationValue $value }}", {{ ParamName $value.Type true 0 }})
	return b
}
{{ end }}

{{ range $key, $value := .PostParams }}
func (b *{{ $.RequestType }}Impl) {{ $key }}({{ ParamsList $value.Type }}) {{ $.RequestType }} {
	b.postBody = {{ ParamName $value.Type false 0 }}
	return b
}
{{ end }}

{{ range $key, $value := .HeaderParams }}
func (b *{{ $.RequestType }}Impl) {{ $key }}({{ ParamsList $value.Type }}) {{ $.RequestType }} {
	b.headerParams["{{ AnnotationValue $value }}"] = {{ ParamName $value.Type true 0 }}
	return b
}
{{ end }}

{{ range $key, $value := .PostMultiPartParams }}
func (b *{{ $.RequestType }}Impl) {{ $key }}({{ ParamsList $value.Type }}) {{ $.RequestType }} {
	b.postMultiPartParams["{{ AnnotationValue $value }}"] = {{ ParamName $value.Type true 0 }}
	return b
}
{{ end }}

func (b *{{ .RequestType }}Impl) applyPathSubstituions(api string) string {
	if len(b.pathSubstitutions) == 0 {
		return api
	}

	for key, value := range b.pathSubstitutions {
		api = strings.Replace(api, "{" + key + "}", value, -1)
	}

	return api
}

func (b *{{ .RequestType }}Impl) build() (req *http.Request, err error) {
	url := b.baseUrl + b.applyPathSubstituions("{{ .ApiEndpoint }}")
	httpMethod := "{{ .HttpMethod }}"
	switch httpMethod {
	case "POST", "PUT":
		if b.postBody != nil {
			// Assume the body is to be marshalled to JSON
			contentBody, err := json.Marshal(b.postBody)
			if err != nil {
				return nil, err
			}
			contentReader := bytes.NewReader(contentBody)
			req, err = http.NewRequest(httpMethod, url, contentReader)
			if err != nil {
				return nil, err
			}
			req.Header.Set("Content-Type", "application/json")
		} else if len(b.postFormParams) > 0 {
			contentForm := b.postFormParams.Encode()
			contentReader := strings.NewReader(contentForm)
			if req, err = http.NewRequest(httpMethod, url, contentReader); err != nil {
				return nil, err
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		} else if len(b.postMultiPartParam) > 0 {
			contentBody := &bytes.Buffer{}
			writer := multipart.NewWriter(contentBody)
			for key, value := range b.postMultiPartParam {
				if err := writer.WriteField(key, string(value)); err != nil {
					return nil, err
				}
			}
			if err = writer.Close(); err != nil {
				return nil, err
			}
			if req, err = http.NewRequest(httpMethod, url, contentBody); err != nil {
				return nil, err
			}
			req.Header.Set("Content-Type", "multipart/form-data")
		}
	case "GET", "DELETE":
		req, err = http.NewRequest(httpMethod, url, nil)
		if err != nil {
			return nil, err
		}
		if len(b.queryParams) > 0 {
			req.URL.RawQuery = b.queryParams.Encode()
		}
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range b.headerParams {
		req.Header.Set(key, value)
	}
	return req, nil
}

{{ if and .ResponseType .SyncResponse }}
func (b *{{ $.RequestType }}Impl) {{ $.SyncResponse | FunctionName }}() ({{ $.ResponseType }}, error) {
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
func (b *{{ $.RequestType }}Impl) {{ $.AsyncResponse | FunctionName }}({{ ParamsList $.AsyncResponse.Type }}) {
	if {{ ParamName $.AsyncResponse.Type false 0 }} != nil {
		{{ ParamName $.AsyncResponse.Type false 0 }}.OnStart()
	}

	go func(b *{{ $.RequestType }}Impl) {
		response, err := b.{{ $.SyncResponse | FunctionName }}()

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

// isInteractive will return true if os.Stdin appears to be interactive
func isInteractive() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fileInfo.Mode()&(os.ModeCharDevice|os.ModeCharDevice) != 0
}

// getFunctionName returns the name of the function
func getFunctionName(f *ast.Field) string {
	return f.Names[0].Name
}

// getAnnotationValue returns the value represented by the annotation in the field's comment
func getAnnotationValue(f *ast.Field) string {
	re := regexp.MustCompile(pattern)
	comment := f.Doc.Text()

	for annotionType, _ := range annotationTypes {
		if key, valid := extractAnnotationValue(re, comment, annotionType); valid {
			return key
		}
	}

	log.Fatalf("Must have query or path or field parameter defined.")
	return ""
}

// getParamName returns the name of the parameter in the field's argument list
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

	param := p.List[index]
	paramName := param.Names[0].Name
	if forceString {
		paramName = "fmt.Sprintf(\"%v\"," + paramName + ")"
	}
	return paramName
}

// getParamsList returns a comma separated list of parameter name, parameter type pairs
// Example: size int8, name string, lat float64
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

// getParamType will return the parameter type
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
	sync               string = "SYNC"
	async              string = "ASYNC"
	header             string = "HEADER"
	path               string = "PATH"
	query              string = "QUERY"
	field              string = "FIELD"
	part               string = "PART"
	httpMethodGet      string = "GET"
	httpMethodPost     string = "POST"
	httpMethodPostForm string = "POST_FORM"
	httpMethodPut      string = "PUT"
	httpMethodDelete   string = "DELETE"
	httpMethodHead     string = "HEAD"

	// pattern represents the annotation regex pattern
	// A valid annotation example is: @GET("/photos/{id}/comments"), where we return
	// ['GET("/photos/{id}/comments")', 'GET', '/photos/{id}/comments']
	pattern string = `@(\w+)\(\"(.*)\"\)`
)

type empty struct{}

var annotationTypes = map[string]empty{
	field:  empty{},
	header: empty{},
	part:   empty{},
	path:   empty{},
	query:  empty{},
	sync:   empty{},
	async:  empty{},
}

var httpMethods = map[string]empty{
	httpMethodDelete:   empty{},
	httpMethodGet:      empty{},
	httpMethodHead:     empty{},
	httpMethodPost:     empty{},
	httpMethodPostForm: empty{},
	httpMethodPut:      empty{},
}

type generateInfo struct {
	Pkg                 string
	RequestType         string
	ApiEndpoint         string
	HttpMethod          string
	PathSubstitutions   map[string]*ast.Field
	QueryParams         map[string]*ast.Field
	PostFormParams      map[string]*ast.Field
	PostMultiPartParams map[string]*ast.Field
	PostParams          map[string]*ast.Field
	HeaderParams        map[string]*ast.Field
	SyncResponse        *ast.Field
	AsyncResponse       *ast.Field
	CallbackType        string
	ResponseType        string
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
			Pkg:                 pkg,
			PathSubstitutions:   make(map[string]*ast.Field),
			QueryParams:         make(map[string]*ast.Field),
			PostFormParams:      make(map[string]*ast.Field),
			PostMultiPartParams: make(map[string]*ast.Field),
			PostParams:          make(map[string]*ast.Field),
			HeaderParams:        make(map[string]*ast.Field),
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
			match := v.re.FindStringSubmatch(f.Doc.List[0].Text)
			if len(match) != 3 {
				continue
			}
			key := match[1]
			value := match[2]
			param := f.Names[0].Name

			switch key {
			case field:
				v.generateInfo.PostFormParams[param] = f
			case header:
				v.generateInfo.HeaderParams[param] = f
			case part:
				v.generateInfo.PostMultiPartParams[param] = f
			case path:
				v.generateInfo.PathSubstitutions[param] = f
			case query:
				v.generateInfo.QueryParams[param] = f
			case sync:
				v.generateInfo.SyncResponse = f
				v.generateInfo.ResponseType = value
			case async:
				v.generateInfo.AsyncResponse = f
				v.generateInfo.CallbackType = value
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

func extractAnnotationValue(re *regexp.Regexp, s, paramType string) (string, bool) {
	match := re.FindStringSubmatch(s)
	if len(match) == 3 && match[1] == paramType {
		return match[2], true
	}
	return "", false
}

func extractHttpMethodTag(re *regexp.Regexp, s string) (string, string) {
	match := re.FindStringSubmatch(s)
	if len(match) == 3 && isAnnotationHttpMethod(match[1]) {
		method := match[1]
		api := match[2]
		if method == httpMethodPostForm {
			method = httpMethodPost
		}
		return method, api
	}
	return "", ""
}

func isAnnotationHttpMethod(s string) bool {
	_, ok := httpMethods[s]
	return ok
}
