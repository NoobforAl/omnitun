# Runs a go command with the project's sing-box build tags (Windows equivalent of the Makefile).
# Usage: ./scripts/go.ps1 test ./...    |    ./scripts/go.ps1 build -o bin/ ./cmd/...
$tags = 'with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api,with_openvpn,with_openconnect,badlinkname,tfogo_checklinkname0'
$cmd = $args[0]
$rest = @($args | Select-Object -Skip 1)
if ($cmd -in 'build', 'test', 'run', 'install') {
    & go $cmd "-tags=$tags" '-ldflags=-checklinkname=0' @rest
} else {
    & go $cmd "-tags=$tags" @rest
}
exit $LASTEXITCODE
