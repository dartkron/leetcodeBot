package tests

import "errors"

// ErrBypassTest Universal special error to send from Mocks to calls
var ErrBypassTest error = errors.New("test bypass error")

// ErrWrongJSON Error about broken JSON for asser.Equal on err.Error()
var ErrWrongJSON = errors.New("unexpected end of JSON input")
