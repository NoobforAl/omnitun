# sing-box feature tags used by every Go build and test.
TAGS := with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api,with_openvpn,with_openconnect,badlinkname,tfogo_checklinkname0
LDFLAGS := -s -w -checklinkname=0
GOFLAGS_ALL := -tags "$(TAGS)" -ldflags "$(LDFLAGS)"

.PHONY: build test vet tor app-test

build:
	go build $(GOFLAGS_ALL) -o bin/ ./cmd/...

test:
	go test $(GOFLAGS_ALL) ./...

vet:
	go vet -tags "$(TAGS)" ./...

tor:
	cd tor && cargo test

app-test:
	cd app && flutter analyze && flutter test
