//go:build with_naive_outbound

// Run with the Naive client included, e.g. on Windows (cronet loads libcronet.dll):
//   go test -tags=with_gvisor,with_quic,with_wireguard,with_utls,with_naive_outbound,with_purego,badlinkname,tfogo_checklinkname0 -ldflags=-checklinkname=0 -run Generated ./server/runtime/

package runtime

import (
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"testing"
)

const naiveIncluded = true

// On Windows cronet loads libcronet.dll from PATH; use the one in the module cache.
func init() {
	if goruntime.GOOS != "windows" {
		return
	}
	out, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		return
	}
	dirs, _ := filepath.Glob(filepath.Join(strings.TrimSpace(string(out)), "github.com", "sagernet", "cronet-go", "lib", "windows_"+goruntime.GOARCH+"@*"))
	if len(dirs) > 0 {
		sort.Strings(dirs)
		os.Setenv("PATH", dirs[len(dirs)-1]+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
}

var _ = testing.Short
