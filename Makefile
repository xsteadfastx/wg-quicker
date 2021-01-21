WIREGUARD-GO_VERSION ?= v0.0.20201118
WIREGUARD-GO_REPO ?= https://git.zx2c4.com/wireguard-go
export GO111MODULE := on
GO ?= go
GOFLAGS ?= -ldflags '-s -w -extldflags "-static"'
GOARM ?= 5
GOARCH ?= amd64

ifeq ($(GOARCH),arm)
	GOARMLINE := GOARM=$(GOARM)
else
	GOARMLINE :=
endif

all: clean wg-quicker

.PHONY: clean
clean:
	rm wg-quicker || true
	rm -rf assets || true
	rm -rf bin || true

.PHONY: wireguard-go
wireguard-go: clean
	mkdir bin
	cd /tmp; \
		git clone $(WIREGUARD-GO_REPO); \
		cd wireguard-go; \
		git checkout -b $(WIREGUARD-GO_VERSION) $(WIREGUARD-GO_VERSION); \
		GOOS=linux GOARCH=$(GOARCH) $(GOARMLINE) $(GO) build -v -o wireguard-go $(GOFLAGS) .
	mv /tmp/wireguard-go/wireguard-go bin/wireguard-go

.PHONY: generate
generate: wireguard-go
	go generate -v ./...

wg-quicker: generate
	$(GO) build -v $(GOFLAGS) -o "$@" cmd/wg-quicker/main.go

.PHONY: build
build:
	goreleaser build --rm-dist --snapshot --parallelism=1

.PHONY: release
release:
	goreleaser release --rm-dist --snapshot --parallelism=1


.PHONY: test
test:
	GOFLAGS=-mod=vendor go test -race -cover -v ./...

.PHONY: lint
lint:
	golangci-lint run --enable-all --disable gomnd --disable godox --disable exhaustivestruct --timeout 5m
