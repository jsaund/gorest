GoREST
======
A type-safe HTTP API implementation tool for Go ( inspired by Square's Retrofit library )

## Installation
```bash
$ go get github.com/jsaund/gorest
```

## Features
GoREST transforms your HTTP API Interface definition in to a type-safe request and response implementation with both synchronous and asynchronous execution functions. Defining the API as an interface provides you the ability to create mock objects and testable code. The request implemenation also maintains combatability with `gomobile` so you can generate HTTP API implementations for your cross-platform SDK written in Go!

The tool is design to accept a `.go` file that has an interface with the support annotations. The tool parses the interface and generates the respective implementation. The Request is generated as a Builder. The builder can be executed using either the `Run` or `RunAsync` function for either synchronous or asynchronous execution, respectively.

### Defining an HTTP API
Each file containing a HTTP API definition should contain the following `go:generate` tag:
```text
//go:generate $GOPATH/src/github.com/jsaund/gorest/gorest -input [NAME OF GO FILE API DEFINITION] -output [NAME OF GO FILE OUTPUT] -pkg [YOUR PACKAGE NAME]
```

#### Request Method
Every interface must have a HTTP annotation that provides the request method and relative URL. There are four supported HTTP method annotations: `GET`, `POST`, `POST_FORM`, `PUT`, `DELETE`.
Example:
```go
// @GET("/photos")
type GetPhotosRequestBuilder interface {
    // ... function declarations for request parameters
}
```

#### URL Manipulation
A request URL can be updated dynamically using replacement blocks and parameters on the method. A replacement block is an alphanumeric string surrounded by `{` and `}`. A corresponding parameter must be annotated with `@PATH` using the same string.
```go
// @GET("/photos/{id}")
type GetPhotoDetailsRequestBuilder interface {
    // @PATH("id")
    PhotoID(id string) GetPhotoDetailsRequestBuilder

    // ... function declarations for request parameters
}
```

#### Query Parameters
In addition to updating a request URL dynamically, you can also supply query parameters using the `@QUERY` annotation.
```go
// @GET("/photos/{id}")
type GetPhotoDetailsRequestBuilder interface {
    // @PATH("id")
    PhotoID(id string) GetPhotoDetailsRequestBuilder

    // @QUERY("image_size")
    ImageSize(size int8) GetPhotoDetailsRequestBuilder

    // @QUERY("comments")
    Comments(include int8) GetPhotoDetailsRequestBuilder
}
```

#### Request Body
To specifcy an object for use as an HTTP request body you must use the `@BODY` annotation. Only one `@BODY` annotation must be used per request. The object must support JSON serialization.
```go
// @POST("/photos")
type PostPhotoRequestBuilder interface {
    // @BODY("photo")
    PhotoMetadata(metadata Metadata) PostPhotoRequestBuilder
}
```

#### Form Encoded
To send form-encoded data you must first use the `@POST_FORM` HTTP annotation for the interface declaration and then declare any key-value pair of form data using the `@FIELD` annotation.
```go
// @POST_FORM("/photos/{id}/comments")
type PostCommentRequestBuilder interface {
	// @PATH("id")
	PhotoID(id string) PostCommentRequestBuilder

	// @FIELD("body")
	Body(body string) PostCommentRequestBuilder
}
```

#### Multipart Data
Multipart requests can be defined with the `@PART` annotation. This is applicable for only `@POST` or `@PUT` operations.
```go
// @POST("/upload")
type PostUploadPhotoRequestBuilder interface {
	// @PART("photo_id")
	PhotoID(id string) PostUploadPhotoRequestBuilder

	// @PART("file")
	File(body string) PostUploadPhotoRequestBuilder
}
```
As of the current version, this operation is considered fairly expensive as it requires copying the entire payload of the part in to memory and marshaling it to the Go SDK.
This will be improved in the future to stream data.

#### Headers
You can also supply custom header key-value pair definitions using the `@HEADER` annotation.
```go
// @GET("/users/{id}/friends")
type GetUserFriendsRequestBuilder interface {
	// @PATH("photo_id")
	PhotoID(id string) GetUserFriendsRequestBuilder

	// @HEADER("User-Agent")
	UserAgent(agent string) GetUserFriendsRequestBuilder
}
```
Note that header names will append to any existing values associated with name.
Supplying the empty string for the header value will remove the header key-value pair from the map.

## Contributors
Contributors wanted!
Please feel free to create an issue for features or improvements or open a pull request with testing.

## License
GoREST is MIT License.
