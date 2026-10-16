package mobile

import "testing"

func TestStartTunnelFromJSONNeedsTheCore(t *testing.T) {
	mu.Lock()
	saved := srv
	srv = nil
	mu.Unlock()
	defer func() {
		mu.Lock()
		srv = saved
		mu.Unlock()
	}()
	if err := StartTunnelFromJSON(`{"profiles":[]}`); err == nil {
		t.Fatal("started without a running core")
	}
}
