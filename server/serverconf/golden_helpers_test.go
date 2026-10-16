package serverconf

import (
	"encoding/json"
	"reflect"
	"strings"
)

func replaceAll(s, old, repl string) string { return strings.ReplaceAll(s, old, repl) }

// jsonEscape is s as it appears inside a JSON string (backslashes doubled).
func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}

func deepEqualJSON(a, b any) bool { return reflect.DeepEqual(a, b) }
