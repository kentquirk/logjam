package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang/gddo/httputil/header"
	"github.com/kentquirk/stringset/v2"
	"github.com/labstack/echo/v4"
)

type malformedRequest struct {
	status int
	msg    string
}

func (mr *malformedRequest) Error() string {
	return mr.msg
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			return &malformedRequest{status: http.StatusUnsupportedMediaType, msg: msg}
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("Request body contains badly-formed JSON")
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			return &malformedRequest{status: http.StatusRequestEntityTooLarge, msg: msg}

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		msg := "Request body must only contain a single JSON object"
		return &malformedRequest{status: http.StatusBadRequest, msg: msg}
	}

	return nil
}

func parseIntWithDefault(input string, def int) (int, error) {
	if input == "" {
		return def, nil
	}

	n, err := strconv.Atoi(input)
	if err != nil {
		return def, echo.NewHTTPError(http.StatusBadRequest, "parameter must be an integer")
	}
	return n, nil
}

// err400 returns 400 and is used to discourage random queries
func err400(c echo.Context) error {
	return c.String(http.StatusBadRequest, "Go away.")
}

// doc returns a documentation page
func doc(c echo.Context) error {
	doctext := `
	<h1>Logjam</h1>
	<p>This service accepts logging requests, and distributes the result to any
	number of configured options</p>
	`
	return c.String(http.StatusOK, doctext)
}

// health returns 200 Ok and can be used by a load balancer to indicate
// that the service is stable
func health(c echo.Context) error {
	return c.String(http.StatusOK, "ok\n")
}

// logOne is intended to be used as a goroutine to do the work of logging.
// It returns nothing.
func logOne(m map[string]interface{}) {
	fieldNames := stringset.New()
	for k := range m {
		fieldNames.Add(k)
	}
	fmt.Printf("FieldNames: %s\n", fieldNames.Join(", "))
}

// logSingle launches a goroutine to do the work of logging.
// It returns nothing.
func logSingle(m map[string]interface{}) {
	go logOne(m)
}

// logSinglePost is a handler that receives a single log entry in the POST body.
func logSinglePost(c echo.Context) error {
	var fields map[string]interface{}

	switch c.Request().Header.Get("content-type") {
	case "application/json", "text/plain":
		err := json.NewDecoder(c.Request().Body).Decode(&fields)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Unable to parse body as JSON")
		}
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "content must be json")
	}

	// do the logging as a goroutine
	logSingle(fields)
	return c.String(http.StatusOK, "ok\n")
}

// logSinglePut is a handler that receives a single log entry as query parameters
func logSinglePut(c echo.Context) error {
	fields := make(map[string]interface{})

	values := c.Request().URL.Query()
	for k := range values {
		fields[k] = values.Get(k)
	}

	logSingle(fields)
	return c.String(http.StatusOK, "ok")
}

// logMulti is a handler that receives a single log entry in the POST body.
func logMulti(c echo.Context) error {
	ary := make([]map[string]interface{}, 0)

	switch c.Request().Header.Get("content-type") {
	case "application/json", "text/plain":
		err := json.NewDecoder(c.Request().Body).Decode(&ary)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Unable to parse body as array of JSON")
		}
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "content must be json")
	}

	// do the logging as a goroutine
	for _, v := range ary {
		logSingle(v)
	}
	return c.String(http.StatusOK, "ok")
}
