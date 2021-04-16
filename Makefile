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
	rm third_party/wireguard-go/wireguard-go || true
	rm assets/wireguard-go/wireguard-go || true

.PHONY: wireguard-go
wireguard-go: clean
	cd third_party/wireguard-go; \
		GOOS=linux GOARCH=$(GOARCH) $(GOARMLINE) $(GO) build -v -o wireguard-go $(GOFLAGS) .
	cp third_party/wireguard-go/wireguard-go assets/wireguard-go

.PHONY: generate
generate:
	go generate ./...

wg-quicker: wireguard-go
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

.PHONY: install-tools
install-tools:
	@cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install -v %
