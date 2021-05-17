package main

import (
	"fmt"
	"github.com/go-chi/render"
	"net/http"
)

type ApiError struct {
	Err        error  `json:"-"`
	StatusCode int    `json:"-"`
	StatusText string `json:"statusText"`
	Message    string `json:"message"`
}

var (
	ApiErrBadRequestRoomNotProvided = &ApiError{
		StatusCode: http.StatusBadRequest,
		StatusText: "room name required",
		Message:    "you must provide a room name",
	}
	ApiErrRoomNotFound = &ApiError{
		StatusCode: http.StatusNotFound,
		StatusText: "room not found",
		Message:    "check the room name and try again",
	}
)

func (e *ApiError) Error() string {
	return fmt.Sprintf("%d %s %s err: %v", e.StatusCode, e.StatusText, e.Message, e.Err)
}

func (e *ApiError) Render(_ http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.StatusCode)
	return nil
}
