APP := lazyports
DIST := dist
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || printf 'none')
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X github.com/DriesVanHool/lazyports/internal/buildinfo.Version=$(VERSION) -X github.com/DriesVanHool/lazyports/internal/buildinfo.Commit=$(COMMIT) -X github.com/DriesVanHool/lazyports/internal/buildinfo.Date=$(DATE)

.PHONY: build test fmt cross-build package-release clean

build:
	go build -ldflags "$(LDFLAGS)" -o $(APP) .

test:
	go test ./...

fmt:
	gofmt -w main.go internal/ports/*.go internal/ui/*.go

cross-build:
	mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(APP)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(APP)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(APP)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(APP)-windows-amd64.exe .

package-release: cross-build
	tar -C $(DIST) -czf $(DIST)/$(APP)-linux-amd64.tar.gz $(APP)-linux-amd64
	tar -C $(DIST) -czf $(DIST)/$(APP)-darwin-amd64.tar.gz $(APP)-darwin-amd64
	tar -C $(DIST) -czf $(DIST)/$(APP)-darwin-arm64.tar.gz $(APP)-darwin-arm64
	zip -j $(DIST)/$(APP)-windows-amd64.zip $(DIST)/$(APP)-windows-amd64.exe
	cd $(DIST) && sha256sum *.tar.gz *.zip > checksums.txt

clean:
	rm -rf $(APP) $(DIST)
