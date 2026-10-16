package control

import (
	"errors"
	"testing"
)

func TestParseExitInfoFormats(t *testing.T) {
	for body, want := range map[string]ExitInfo{
		`{"ip":"203.0.113.5","country":"de"}`:                            {IP: "203.0.113.5", Country: "DE"},
		`{"ip":"2001:db8::1","success":true,"country_code":"NL"}`:        {IP: "2001:db8::1", Country: "NL"},
		`{"query":"198.51.100.7","status":"success","countryCode":"SE"}`: {IP: "198.51.100.7", Country: "SE"},
		`{"IsTor":true,"IP":"185.220.101.1"}`:                            {IP: "185.220.101.1"},
		`{"ip":"192.0.2.1","country":"Germany"}`:                         {IP: "192.0.2.1"},     // a name is not a code
		`{"ip":"192.42.116.51","country":"T1"}`:                          {IP: "192.42.116.51"}, // country.is says T1 for Tor exits
		`{"ip":"192.0.2.2","country":"XX"}`:                              {IP: "192.0.2.2"},
	} {
		got, err := parseExitInfo([]byte(body))
		if err != nil || *got != want {
			t.Errorf("%s: %+v, %v; want %+v", body, got, err, want)
		}
	}
	for _, body := range []string{``, `<html>`, `{"country":"DE"}`, `[1,2]`, `{"ip":""}`} {
		if _, err := parseExitInfo([]byte(body)); err == nil {
			t.Errorf("%q was accepted", body)
		}
	}
}

// Without a tunnel the lookup refuses: an answer would be about this device, not the VPN.
func TestExitInfoNeedsATunnel(t *testing.T) {
	s, addr := newTestServer(t, ServerConfig{})
	c, err := Dial(t.Context(), addr, s.Token())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var info ExitInfo
	err = c.Call(t.Context(), "exit_info", ExitInfoParams{}, &info)
	var ce *Error
	if !errors.As(err, &ce) || ce.Code != CodeNotRunning {
		t.Fatalf("err = %v, want NOT_RUNNING", err)
	}
}

func errorsAs(err error, target **Error) bool { return errors.As(err, target) }
