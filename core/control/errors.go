package control

import (
	"errors"
	"strconv"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/subscription"
)

// badRequest is the error reply for a request whose values are wrong; an
// editor's problems keep their code and arguments, so the UI can translate them.
func badRequest(err error) *Error {
	if fe, ok := errors.AsType[*profile.FormError](err); ok {
		return &Error{Code: fe.Code, Message: err.Error(), Args: fe.Args}
	}
	return &Error{Code: CodeBadRequest, Message: err.Error()}
}

// importError is the error reply for text that could not be imported.
func importError(err error) *Error {
	e := &Error{Code: CodeImportFailed, Message: err.Error()}
	var ue *importer.UnsupportedError
	switch {
	case errors.Is(err, importer.ErrXrayServer):
		e.Code = CodeImportXrayServer
	case errors.Is(err, importer.ErrNoProfiles):
		e.Code = CodeImportNothing
	case errors.As(err, &ue):
		e.Code = CodeImportUnsupported
		e.Args = map[string]string{"scheme": ue.Scheme}
	case errors.Is(err, importer.ErrUnsupported):
		e.Code = CodeImportUnsupported
	}
	return e
}

// subscriptionError is the error reply for a subscription that could not be fetched.
func subscriptionError(err error) *Error {
	e := &Error{Code: CodeSubscriptionFailed, Message: err.Error()}
	var he *subscription.HTTPError
	switch {
	case errors.Is(err, subscription.ErrAuth):
		e.Code = CodeAuthFailed
	case errors.As(err, &he):
		e.Code = CodeSubHTTP
		e.Args = map[string]string{"status": strconv.Itoa(he.Status)}
	case errors.Is(err, subscription.ErrTooLarge):
		e.Code = CodeSubTooLarge
	case errors.Is(err, subscription.ErrBadURL):
		e.Code = CodeSubBadURL
	}
	return e
}
