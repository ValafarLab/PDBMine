package main

import (
	"net/http"

	//Chi Router
	"github.com/go-chi/render"
)

//ErrResponse is a structure for return errors
type ErrResponse struct {
	Err            error  `json:"-"`               // low-level runtime error
	HTTPStatusCode int    `json:"-"`               // http response status code
	StatusText     string `json:"status"`          // user-level status message
	AppCode        int64  `json:"code,omitempty"`  // application-specific error code
	ErrorText      string `json:"error,omitempty"` // application-level error message, for debugging
}

//ErrInvalidRequest returns an invalid request response.
func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid request",
		ErrorText:      err.Error(),
	}
}

//ErrNotFound returns a record not found response.
func ErrNotFound(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 404,
		StatusText:     "Record Not Found",
		ErrorText:      err.Error(),
	}
}

//ErrCannotRetrieve returns a 503 when Redis is not available
func ErrCannotRetrieve(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 503,
		StatusText:     "Unable to save to or retrieve from Redis",
		ErrorText:      err.Error(),
	}
}

//ErrCannotGenerateUID returns a 503 when a UID cannot be created
func ErrCannotGenerateUID(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 503,
		StatusText:     "Unable to generate a unique ID",
		ErrorText:      err.Error(),
	}
}

//ErrCannotSaveToDisk returns a 503 when the file cannot be saved to disk
func ErrCannotSaveToDisk(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 503,
		StatusText:     "Unable to convert to string",
		ErrorText:      err.Error(),
	}
}

//ErrCannotRetrieveFromDisk returns a 404 when the file cannot be saved to disk
func ErrCannotRetrieveFromDisk(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 404,
		StatusText:     "Unable to read data",
		ErrorText:      err.Error(),
	}
}

//Render renders an ErrResponse
func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}
