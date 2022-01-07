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

GORELEASER := $(GO) run github.com/goreleaser/goreleaser
GOLANGCI_LINT := $(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint

all: clean wg-quicker

.PHONY: clean
clean:
	rm wg-quicker || true
	rm third_party/wireguard-go/wireguard-go || true
	rm assets/wireguard-go/wireguard-go || true

.PHONY: wireguard-go
wireguard-go: clean
	cd third_party/wireguard-go; \
		GOOS=linux GOARCH=$(GOARCH) $(GOARMLINE) $(GO) build -v -o wireguard-go $(GOFLAGS) .
	cp third_party/wireguard-go/wireguard-go assets/wireguard-go

.PHONY: generate
generate:
	$(GO) generate ./...

wg-quicker: wireguard-go
	$(GO) build -v $(GOFLAGS) -o "$@" cmd/wg-quicker/main.go

.PHONY: build
build:
	$(GORELEASER) build --rm-dist --snapshot --parallelism=1

.PHONY: release
release:
	$(GORELEASER) release --rm-dist --snapshot --parallelism=1


.PHONY: test
test:
	GOFLAGS=-mod=vendor $(GO) test -race -cover -v ./...

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run \
		--enable-all \
		--disable gomnd \
		--disable godox \
		--disable exhaustivestruct \
		--disable varnamelen \
		--timeout 5m

.PHONY: tidy
tidy:
	$(GO) mod tidy
	$(GO) mod vendor


.PHONY: install-tools
install-tools:
	$(GO) list -f '{{range .Imports}}{{.}} {{end}}' third_party/tools/tools.go | xargs go install -v
