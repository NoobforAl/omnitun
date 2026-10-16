package engine

// Connection stages, reported in State.Progress so the UI can show a
// 0-100 % ring and say exactly where a connection is or where it failed.
// "permission" (5 %) happens in the UI before the core is asked to connect.
const (
	StageNetworkCheck = "network_check"
	StagePrepare      = "prepare"
	StageEngineStart  = "engine_start"
	StageTunnelUp     = "tunnel_up"
	StageHandshake    = "handshake"
	StageConnected    = "connected"
)

var stagePercent = map[string]int{
	StageNetworkCheck: 15,
	StagePrepare:      25,
	StageEngineStart:  45,
	StageTunnelUp:     65,
	StageHandshake:    85,
	StageConnected:    100,
}

// stageLabel is used in error messages ("<label> failed: ...").
var stageLabel = map[string]string{
	StageNetworkCheck: "Checking internet connection",
	StagePrepare:      "Preparing configuration",
	StageEngineStart:  "Starting engine",
	StageTunnelUp:     "Opening tunnel",
	StageHandshake:    "Contacting server",
	StageConnected:    "Connected",
}

// Progress is one connection stage.
type Progress struct {
	Stage   string `json:"stage"`
	Percent int    `json:"percent"`
	// Profile is the display name of the profile being tried, if relevant.
	Profile string `json:"profile,omitempty"`
	// Detail is a finer-grained status, e.g. Tor's bootstrap step.
	Detail string `json:"detail,omitempty"`
	// DetailCode names Detail for translation (the Detail* constants); DetailN
	// and DetailM are the numbers in it (percent; attempt and attempts).
	DetailCode string `json:"detail_code,omitempty"`
	DetailN    int    `json:"detail_n,omitempty"`
	DetailM    int    `json:"detail_m,omitempty"`
}

// Codes of the finer-grained statuses.
const (
	DetailTorStarting  = "TOR_STARTING"
	DetailTorBootstrap = "TOR_BOOTSTRAP" // N is the percent
	DetailTorCircuit   = "TOR_CIRCUIT"
	DetailTorRetry     = "TOR_RETRY" // N of M
)

func progress(stage, profile string) *Progress {
	return &Progress{Stage: stage, Percent: stagePercent[stage], Profile: profile}
}

// ReportProgress publishes a stage while not yet running, e.g. the network
// check the control server does before Start.
func (e *Engine) ReportProgress(stage string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.instance != nil {
		return
	}
	e.setState(State{Status: StatusStarting, Progress: progress(stage, "")})
}

// FailAt reports a failure at a stage that happens outside the engine.
func (e *Engine) FailAt(stage string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.instance != nil {
		return
	}
	e.setState(State{Status: StatusError, Error: stageLabel[stage] + " failed: " + err.Error(), Progress: progress(stage, "")})
}
