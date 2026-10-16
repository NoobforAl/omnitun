package control

import (
	"errors"
	"fmt"
	"testing"

	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/profile"
	"github.com/NoobforAl/omnitun/core/subscription"
)

func TestEditorProblemsKeepTheirCodeAndArguments(t *testing.T) {
	_, err := profile.FromValues("trojan", map[string]string{"server": "example.com"})
	e := badRequest(err)
	if e.Code != profile.FormRequired || e.Args["label"] == "" || e.Message == "" {
		t.Fatalf("missing field: %+v", e)
	}
	_, err = profile.FromValues("trojan", map[string]string{"server": "example.com", "port": "abc", "password": "x"})
	if e := badRequest(err); e.Code != profile.FormNotNumber || e.Args["label"] != "Port" {
		t.Fatalf("bad number: %+v", e)
	}
	// Anything else stays a plain bad request.
	if e := badRequest(errors.New("something else")); e.Code != CodeBadRequest || e.Args != nil {
		t.Fatalf("other: %+v", e)
	}
	if e := badRequest(fmt.Errorf("wrapped: %w", err)); e.Code != profile.FormNotNumber {
		t.Fatalf("a wrapped editor problem lost its code: %+v", e)
	}
}

func TestImportFailuresHaveCodes(t *testing.T) {
	cases := []struct {
		text, code, scheme string
	}{
		{"   ", CodeImportUnsupported, ""},
		{"hello world", CodeImportNothing, ""},
		{"nope://example.com", CodeImportNothing, ""},
		{"vless://not-valid", CodeImportNothing, ""},
		{`{"inbounds":[{"port":443,"protocol":"vless","settings":{"clients":[{"id":"x"}]}}],"outbounds":[]}`, CodeImportXrayServer, ""},
	}
	for _, c := range cases {
		_, err := importer.Import(c.text)
		if err == nil {
			t.Errorf("%q imported", c.text)
			continue
		}
		e := importError(err)
		if e.Code != c.code || e.Args["scheme"] != c.scheme {
			t.Errorf("%q: %+v (%v)", c.text, e, err)
		}
	}
}

func TestUnsupportedLinkNamesItsScheme(t *testing.T) {
	_, err := importer.ParseLink("nope://example.com")
	if e := importError(err); e.Code != CodeImportUnsupported || e.Args["scheme"] != "nope" {
		t.Fatalf("%+v (%v)", e, err)
	}
}

func TestSubscriptionFailuresHaveCodes(t *testing.T) {
	for _, c := range []struct {
		err    error
		code   string
		status string
	}{
		{fmt.Errorf("%w (HTTP 401)", subscription.ErrAuth), CodeAuthFailed, ""},
		{&subscription.HTTPError{Status: 502}, CodeSubHTTP, "502"},
		{subscription.ErrTooLarge, CodeSubTooLarge, ""},
		{fmt.Errorf("%w %q", subscription.ErrBadURL, "x"), CodeSubBadURL, ""},
		{errors.New("download subscription: dial tcp: refused"), CodeSubscriptionFailed, ""},
	} {
		e := subscriptionError(c.err)
		if e.Code != c.code || e.Args["status"] != c.status {
			t.Errorf("%v: %+v", c.err, e)
		}
	}
}
