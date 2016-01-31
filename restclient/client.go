package restclient

import (
	"log"
	"net/http"
	"net/http/httputil"
)

const (
	requestTag  = ">>>>>>>>>>>>>>>>>>>> REQUEST"
	responseTag = "<<<<<<<<<<<<<<<<<<<< RESPONSE"
)

// Client provides the RequestBuilder with a configured http.Client object. In addition to a
// http.Client, RequestBuilder can also utilize relative API URLs when the base URL is present.
// Debugging requests and responses can be made possible by enabling the Debug mode to true.
type Client interface {
	BaseURL() string
	Debug() bool
	HttpClient() *http.Client
}

func DebugRequest(request *http.Request) {
	data, err := httputil.DumpRequestOut(request, true)
	logDebugOutput(requestTag, data, err)
}

func DebugResponse(response *http.Response) {
	data, err := httputil.DumpResponse(response, true)
	logDebugOutput(responseTag, data, err)
}

func logDebugOutput(tag string, data []byte, err error) {
	if err == nil {
		log.Printf("%s\n%s\n====================", tag, data)
	} else {
		log.Printf("%s\n%v\n====================", tag, err)
	}
}
