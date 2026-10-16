//go:build with_quic && with_utls && with_wireguard && with_gvisor

package runtime

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	goruntime "runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/NoobforAl/omnitun/core/engine"
	"github.com/NoobforAl/omnitun/core/importer"
	"github.com/NoobforAl/omnitun/core/singbox"
	"github.com/NoobforAl/omnitun/server/serverconf"
	"github.com/NoobforAl/omnitun/server/store"
)

// sing-box empties its list of closed connections at every garbage
// collection. Traffic of a connection that closed shortly before one must
// still be counted: the collector reads the close events, not only the list.
func TestClosedConnectionIsCountedEvenIfAGCEmptiesTheList(t *testing.T) {
	// No tick will run during the test: only the close event can count it.
	oldInterval, oldEvery := accountingInterval, enforceEvery
	accountingInterval, enforceEvery = time.Hour, 1000
	defer func() { accountingInterval, enforceEvery = oldInterval, oldEvery }()

	s, srv := setup(t)
	st := loadState(t, s)

	const payload = 200 << 10
	web := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, strings.Repeat("x", payload))
	}))
	web.Listener.Close()
	var err error
	if web.Listener, err = net.Listen("tcp", net.JoinHostPort(lanIPv4(t), "0")); err != nil {
		t.Fatal(err)
	}
	web.Start()
	defer web.Close()

	var trojan string
	for _, l := range serverconf.Links(st, s, st.User("alice")) {
		if l.Protocol == store.Trojan {
			trojan = l.URL
		}
	}
	res, err := importer.Import(trojan)
	if err != nil {
		t.Fatal(err)
	}
	p := res.Profiles[0]
	proxyPort := freeTCP(t)
	e := engine.New(engine.Config{DataDir: t.TempDir()})
	if err := e.Start(p.AsSlice(), singbox.Options{
		ListenPort: proxyPort, LogLevel: "error", ProxyPrivate: true, Selected: p.ID, TestURL: web.URL + "/probe",
	}); err != nil {
		t.Fatal(err)
	}
	defer e.Stop()
	waitFor(t, "client to connect", 15*time.Second, func() bool { return e.State().Status == engine.StatusRunning })

	proxy, _ := url.Parse("socks5://127.0.0.1:" + strconv.Itoa(proxyPort))
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxy), DisableKeepAlives: true}, Timeout: 10 * time.Second}
	resp, err := client.Get(web.URL + "/data")
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// What a busy server does all the time: collect garbage, which empties
	// sing-box's list of closed connections.
	for range 3 {
		goruntime.GC()
		time.Sleep(20 * time.Millisecond)
	}
	waitFor(t, "the closed connection to be counted", 5*time.Second, func() bool {
		return srv.tracker.MonthTotal("alice") >= payload
	})
}
