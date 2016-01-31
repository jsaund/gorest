package parse

import (
	"go/ast"
	"go/types"
	"regexp"
)

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

var re *regexp.Regexp = regexp.MustCompile(pattern)

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

type Annotation struct {
	Key   string
	Value string
}

type annotationFilter func(key string) bool

type empty struct{}

type ParseResult struct {
	PackageName         string
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

func newParseResult(pkg string) *ParseResult {
	return &ParseResult{
		PackageName:         pkg,
		PathSubstitutions:   make(map[string]*ast.Field),
		QueryParams:         make(map[string]*ast.Field),
		PostFormParams:      make(map[string]*ast.Field),
		PostMultiPartParams: make(map[string]*ast.Field),
		PostParams:          make(map[string]*ast.Field),
		HeaderParams:        make(map[string]*ast.Field),
	}
}

type Parser struct {
	file         *ast.File
	info         *types.Info
	result       *ParseResult
	buildRequest bool
}

func NewParser(file *ast.File, pkg string) *Parser {
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	return &Parser{
		file:   file,
		info:   info,
		result: newParseResult(pkg),
	}
}

func (p *Parser) Parse() *ParseResult {
	ast.Walk(p, p.file)
	return p.result
}

func (p *Parser) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return p
	}

	switch node.(type) {
	case *ast.File:
		// At the start of a file
		// Reset builder flags
		p.buildRequest = false
		break
	case *ast.TypeSpec:
		// Check if we are at the beginning of a request builder declaration
		// or a response / callback declaration
		// This must be an interface
		typeSpec := node.(*ast.TypeSpec)
		switch typeSpec.Type.(type) {
		case *ast.InterfaceType:
			if p.buildRequest {
				p.result.RequestType = typeSpec.Name.Name
			} else {
				p.buildRequest = false
			}
			break
		}
		break
	case *ast.InterfaceType:
		if !p.buildRequest {
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
			annotation, valid := ExtractRequestAnnotation(f.Doc.List[0].Text)
			if !valid {
				continue
			}
			param := f.Names[0].Name

			switch annotation.Key {
			case field:
				p.result.PostFormParams[param] = f
			case header:
				p.result.HeaderParams[param] = f
			case part:
				p.result.PostMultiPartParams[param] = f
			case path:
				p.result.PathSubstitutions[param] = f
			case query:
				p.result.QueryParams[param] = f
			case sync:
				p.result.SyncResponse = f
				p.result.ResponseType = annotation.Value
			case async:
				p.result.AsyncResponse = f
				p.result.CallbackType = annotation.Value
			}
		}
		break
	case *ast.Comment:
		comment := node.(*ast.Comment)
		if annotation, valid := ExtractHttpAnnotation(comment.Text); valid {
			// Extract the HTTP Method and API from the Interface declaration
			p.buildRequest = true
			p.result.HttpMethod = annotation.Key
			p.result.ApiEndpoint = annotation.Value
		}
		break
	}

	return p
}

func httpAnnotationFilter(s string) bool {
	_, ok := httpMethods[s]
	return ok
}

func requestAnnotationFilter(s string) bool {
	_, ok := annotationTypes[s]
	return ok
}

func ExtractHttpAnnotation(s string) (Annotation, bool) {
	annotation, valid := extractAnnotation(httpAnnotationFilter, s)
	if annotation.Key == httpMethodPostForm {
		annotation.Key = httpMethodPost
	}
	return annotation, valid
}

func ExtractRequestAnnotation(s string) (Annotation, bool) {
	return extractAnnotation(requestAnnotationFilter, s)
}

func extractAnnotation(filter annotationFilter, s string) (Annotation, bool) {
	annotation := Annotation{}
	valid := false
	match := re.FindStringSubmatch(s)
	if len(match) == 3 && filter(match[1]) {
		annotation = Annotation{
			Key:   match[1],
			Value: match[2],
		}
		valid = true
	}
	return annotation, valid
}
