package tests

import "errors"

// ErrBypassTest Universal special error to send from Mocks to calls
var ErrBypassTest error = errors.New("test bypass error")
