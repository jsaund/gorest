package generate

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"log"
	"text/template"

	"github.com/jsaund/gorest/parse"
)

var funcMap = template.FuncMap{
	"ParamsList":      getParamsList,
	"ParamName":       getParamName,
	"AnnotationValue": getAnnotationValue,
	"FunctionName":    getFunctionName,
}

// Generate generates the implementation using the details contained in ParseResult.
func Generate(r *parse.ParseResult) ([]byte, error) {
	var builderTemplate = template.Must(template.New("builder").Funcs(funcMap).Parse(`/*
* CODE GENERATED AUTOMATICALLY WITH GOREST (github.com/jsaund/gorest)
* THIS FILE SHOULD NOT BE EDITED BY HAND
*/

package {{.PackageName}}

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/jsaund/gorest/restclient"
)

{{ if .CallbackType }}
type {{ $.CallbackType }} interface {
	OnStart()
	OnError(reason string)
	OnSuccess(response {{ $.ResponseType }})
}
{{ end }}

type {{ .RequestType }}Impl struct {
	pathSubstitutions  map[string]string
	queryParams        url.Values
	postFormParams     url.Values
	postBody           interface{}
	postMultiPartParam map[string][]byte
	headerParams       map[string]string
}

func New{{ .RequestType }}() {{ .RequestType }} {
	return &{{ .RequestType }}Impl{
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
	restClient := restclient.GetClient()
	if restClient == nil {
		return nil, fmt.Errorf("A rest client has not been registered yet. You must call client.RegisterClient first")
	}
	url := restClient.BaseURL() + b.applyPathSubstituions("{{ .ApiEndpoint }}")
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

	restClient := restclient.GetClient()
	if restClient == nil {
		return nil, fmt.Errorf("A rest client has not been registered yet. You must call client.RegisterClient first")
	}

	if restClient.Debug() {
		restclient.DebugRequest(request)
	}

	response, err := restClient.HttpClient().Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	if restClient.Debug() {
		restclient.DebugResponse(response)
	}

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

// getFunctionName returns the name of the function
func getFunctionName(f *ast.Field) string {
	return f.Names[0].Name
}

// getAnnotationValue returns the value represented by the annotation in the field's comment
func getAnnotationValue(f *ast.Field) string {
	comment := f.Doc.Text()
	if annotation, valid := parse.ExtractRequestAnnotation(comment); valid {
		return annotation.Value
	}
	log.Fatalf("Must have query or path or field parameter defined.")
	return ""
}

// getParamName returns the name of the parameter in the field's argument list
func getParamName(function *ast.FuncType, forceString bool, index int) string {
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
func getParamsList(function *ast.FuncType) string {
	p := function.Params
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
