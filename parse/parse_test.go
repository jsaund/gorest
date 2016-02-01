package parse

import (
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
