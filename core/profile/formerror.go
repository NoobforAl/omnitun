package profile

import "fmt"

// Codes of the problems an editor's values can have. The message is English;
// a UI that has a translation for the code uses it with Args.
const (
	FormRequired           = "FORM_REQUIRED"             // args: label
	FormNotNumber          = "FORM_NOT_NUMBER"           // args: label
	FormNotBool            = "FORM_NOT_BOOL"             // args: label
	FormBadChoice          = "FORM_BAD_CHOICE"           // args: label, value, choices
	FormRealityOnlyVLESS   = "FORM_REALITY_ONLY_VLESS"   //
	FormUnknownSecurity    = "FORM_UNKNOWN_SECURITY"     // args: value
	FormReservedBytes      = "FORM_RESERVED_BYTES"       //
	FormRealityKeyRequired = "FORM_REALITY_KEY_REQUIRED" //
)

// FormError is a problem with what was entered in an editor.
type FormError struct {
	Code string
	Args map[string]string
	msg  string
}

func (e *FormError) Error() string { return e.msg }

func formErr(code, format string, a ...any) *FormError {
	return &FormError{Code: code, msg: fmt.Sprintf(format, a...)}
}

func (e *FormError) arg(key, value string) *FormError {
	if e.Args == nil {
		e.Args = map[string]string{}
	}
	e.Args[key] = value
	return e
}
