package parse

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractHttpAnnotation(t *testing.T) {
	nilAnnotaiton := Annotation{}

	type result struct {
		annotation Annotation
		valid      bool
	}

	var testCases = []struct {
		input  string
		output result
	}{
		{
			"@DELETE(\"/test\")",
			result{
				Annotation{"DELETE", "/test"},
				true,
			},
		},
		{
			"@GET(\"/test\")",
			result{
				Annotation{"GET", "/test"},
				true,
			},
		},
		{
			"@HEAD(\"/test\")",
			result{
				Annotation{"HEAD", "/test"},
				true,
			},
		},
		{
			"@POST(\"/test\")",
			result{
				Annotation{"POST", "/test"},
				true,
			},
		},
		{
			"@POST_FORM(\"/test\")",
			result{
				Annotation{"POST", "/test"},
				true,
			},
		},
		{
			"@PUT(\"/test\")",
			result{
				Annotation{"PUT", "/test"},
				true,
			},
		},
		{
			"@INVALID(\"/test\")",
			result{
				nilAnnotaiton,
				false,
			},
		},
		{
			"@SYNC(\"/test\")",
			result{
				nilAnnotaiton,
				false,
			},
		},
		{
			"@QUERY(\"/test\")",
			result{
				nilAnnotaiton,
				false,
			},
		},
		{
			"@(\"/test\")",
			result{
				nilAnnotaiton,
				false,
			},
		},
		{
			"@@(\"/test\")",
			result{
				nilAnnotaiton,
				false,
			},
		},
		{
			"@GET(\"\")",
			result{
				Annotation{"GET", ""},
				true,
			},
		},
	}

	for _, tc := range testCases {
		annotation, valid := ExtractHttpAnnotation(tc.input)
		assert.Equal(t, tc.output.annotation, annotation)
		assert.Equal(t, tc.output.valid, valid)
	}
}

func TestExtractRequestAnnotation(t *testing.T) {
	nilAnnotaiton := Annotation{}

	type result struct {
		annotation Annotation
		valid      bool
	}

	var testCases = []struct {
		input  string
		output result
	}{
		{
			"@FIELD(\"test_1\")",
			result{
				Annotation{"FIELD", "test_1"},
				true,
			},
		},
		{
			"@HEADER(\"test_2\")",
			result{
				Annotation{"HEADER", "test_2"},
				true,
			},
		},
		{
			"@PART(\"test_3\")",
			result{
				Annotation{"PART", "test_3"},
				true,
			},
		},
		{
			"@PATH(\"test_4\")",
			result{
				Annotation{"PATH", "test_4"},
				true,
			},
		},
		{
			"@QUERY(\"test_5\")",
			result{
				Annotation{"QUERY", "test_5"},
				true,
			},
		},
		{
			"@SYNC(\"test_6\")",
			result{
				Annotation{"SYNC", "test_6"},
				true,
			},
		},
		{
			"@ASYNC(\"test_7\")",
			result{
				Annotation{"ASYNC", "test_7"},
				true,
			},
		},
		{
			"@HEAD(\"/test\")",
			result{
				nilAnnotaiton,
				false,
			},
		},
		{
			"@field(\"invalid\")",
			result{
				nilAnnotaiton,
				false,
			},
		},
	}

	for _, tc := range testCases {
		annotation, valid := ExtractRequestAnnotation(tc.input)
		assert.Equal(t, tc.output.annotation, annotation)
		assert.Equal(t, tc.output.valid, valid)
	}
}

func TestParseInvalidCases(t *testing.T) {
	type testCase struct {
		pkg string
		src string
	}

	var testCases = []struct {
		input  testCase
		output *ParseResult
	}{
		// Empty file
		{
			testCase{
				"main",
				`
				package main
				`,
			},
			newParseResult(""),
		},
		// File containing nothing to parse
		{
			testCase{
				"main",
				`
				package main
				func main() {
					println("Hello, World!")
				}
				`,
			},
			newParseResult("main"),
		},
		// File containing interface with missing http annotaiton
		{
			testCase{
				"test",
				`
				package test
				type InvalidRequest interface {
				}
				`,
			},
			newParseResult("test"),
		},
		// Invalid annotation
		{
			testCase{
				"test",
				`
				package test
				// @INVALID("invalid")
				type InvalidRequest interface {
				}
				`,
			},
			newParseResult("test"),
		},
		// Valid http request annotation
		{
			testCase{
				"test",
				`
				package test
				// @GET("/photos")
				type GetPhotosRequestBuilder interface {
				}
				`,
			},
			&ParseResult{
				PackageName:         "test",
				PathSubstitutions:   make(map[string]*ast.Field),
				QueryParams:         make(map[string]*ast.Field),
				PostFormParams:      make(map[string]*ast.Field),
				PostMultiPartParams: make(map[string]*ast.Field),
				PostParams:          make(map[string]*ast.Field),
				HeaderParams:        make(map[string]*ast.Field),
				RequestType:         "GetPhotosRequestBuilder",
				ApiEndpoint:         "/photos",
				HttpMethod:          "GET",
			},
		},
		// Invalid request annotation
		{
			testCase{
				"test",
				`
				package test
				// @GET("/photos")
				type GetPhotosRequestBuilder interface {
					// @INVALID("invalid")
					InvalidOp(i int) GetPhotosRequestBuilder
				}
				`,
			},
			&ParseResult{
				PackageName:         "test",
				PathSubstitutions:   make(map[string]*ast.Field),
				QueryParams:         make(map[string]*ast.Field),
				PostFormParams:      make(map[string]*ast.Field),
				PostMultiPartParams: make(map[string]*ast.Field),
				PostParams:          make(map[string]*ast.Field),
				HeaderParams:        make(map[string]*ast.Field),
				RequestType:         "GetPhotosRequestBuilder",
				ApiEndpoint:         "/photos",
				HttpMethod:          "GET",
			},
		},
	}

	for _, tc := range testCases {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "input.go", tc.input.src, parser.ParseComments)
		assert.NoError(t, err)
		p := NewParser(f, "")
		result := p.Parse()
		assert.ObjectsAreEqualValues(tc.output, result)
	}
}

func TestParseValidCases(t *testing.T) {
	// Valid Request
	src := `
		package test
		// @GET("/photos/{id}")
		type GetPhotoDetailsRequestBuilder interface {
			// @PATH("id")
			PhotoID(id string) GetPhotoDetailsRequestBuilder

			// @QUERY("image_size")
			ImageSize(size int) GetPhotoDetailsRequestBuilder

			// @FIELD("body")
			Body(body string) GetPhotoDetailsRequestBuilder

			// @HEADER("x-type")
			Type(t string) GetPhotoDetailsRequestBuilder

			// @PART("data")
			Data(d []byte) GetPhotoDetailsRequestBuilder

			// @SYNC("GetPhotoDetailsResponse")
			Run() (GetPhotoDetailsResponse, error)

			// @ASYNC("GetPhotoDetailsCallback")
			RunAsync(callback GetPhotoDetailsCallback)
		}
		`
	expectedResult := &ParseResult{
		PackageName:         "test",
		PathSubstitutions:   make(map[string]*ast.Field),
		QueryParams:         make(map[string]*ast.Field),
		PostFormParams:      make(map[string]*ast.Field),
		PostMultiPartParams: make(map[string]*ast.Field),
		PostParams:          make(map[string]*ast.Field),
		HeaderParams:        make(map[string]*ast.Field),
		RequestType:         "GetPhotoDetailsRequestBuilder",
		ApiEndpoint:         "/photos/{id}",
		HttpMethod:          "GET",
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "input.go", src, parser.ParseComments)
	assert.NoError(t, err)

	interfaceDecl := f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.InterfaceType)
	expectedResult.PathSubstitutions["id"] = interfaceDecl.Methods.List[0]
	expectedResult.QueryParams["image_size"] = interfaceDecl.Methods.List[1]
	expectedResult.PostFormParams["body"] = interfaceDecl.Methods.List[2]
	expectedResult.HeaderParams["x-type"] = interfaceDecl.Methods.List[3]
	expectedResult.PostMultiPartParams["data"] = interfaceDecl.Methods.List[4]
	expectedResult.SyncResponse = interfaceDecl.Methods.List[5]
	expectedResult.AsyncResponse = interfaceDecl.Methods.List[6]
	p := NewParser(f, "")
	actualResult := p.Parse()
	assert.ObjectsAreEqualValues(expectedResult, actualResult)
}
