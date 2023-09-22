VERSION = $(shell grep -Po 'v[0-9]+\.[0-9]+\.[0-9]+' version.go)
GITCOMMIT=$(shell git rev-parse --short HEAD)
BUILDTIME=$(shell date -u --iso-8601=seconds)

# Strip debug info
GO_LDFLAGS = -s -w

# Add some build information
GO_LDFLAGS += -X 'main.commit=$(GITCOMMIT)'
GO_LDFLAGS += -X 'main.date=$(BUILDTIME)'

.PHONY: binary
binary: build/serverpilot-tools

build/serverpilot-tools:
	go build -ldflags "$(GO_LDFLAGS)" -o $@

.PHONY: platform-all
platform-all: platform-linux platform-darwin platform-windows

.PHONY: platform-linux
platform-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o build/serverpilot-tools-linux-amd64/serverpilot-tools
	GOOS=linux GOARCH=386 go build -ldflags "$(GO_LDFLAGS)" -o build/serverpilot-tools-linux-386/serverpilot-tools
	GOOS=linux GOARCH=arm64 go build -ldflags "$(GO_LDFLAGS)" -o build/serverpilot-tools-linux-arm64/serverpilot-tools

.PHONY: platform-darwin
platform-darwin:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o build/serverpilot-tools-darwin-amd64/serverpilot-tools
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(GO_LDFLAGS)" -o build/serverpilot-tools-darwin-arm64/serverpilot-tools

.PHONY: platform-windows
platform-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o build/serverpilot-tools-windows-amd64/serverpilot-tools.exe
	GOOS=windows GOARCH=386 go build -ldflags "$(GO_LDFLAGS)" -o build/serverpilot-tools-windows-386/serverpilot-tools.exe
	GOOS=windows GOARCH=arm64 go build -ldflags "$(GO_LDFLAGS)" -o build/serverpilot-tools-windows-arm64/serverpilot-tools.exe

.PHONY: release
release: platform-linux platform-darwin platform-windows
	mkdir build/release
	cd build && tar -zcf release/serverpilot-tools-$(VERSION)-Linux_x86_64.tar.gz serverpilot-tools-linux-amd64
	cd build && tar -zcf release/serverpilot-tools-$(VERSION)-Linux_i386.tar.gz serverpilot-tools-linux-386
	cd build && tar -zcf release/serverpilot-tools-$(VERSION)-Linux_arm64.tar.gz serverpilot-tools-linux-arm64
	cd build && tar -zcf release/serverpilot-tools-$(VERSION)-Darwin_x86_64.tar.gz serverpilot-tools-darwin-amd64
	cd build && tar -zcf release/serverpilot-tools-$(VERSION)-Darwin_arm64.tar.gz serverpilot-tools-darwin-arm64
	cd build && zip -r release/serverpilot-tools-$(VERSION)-Windows_x86_64.zip serverpilot-tools-windows-amd64
	cd build && zip -r release/serverpilot-tools-$(VERSION)-Windows_i386.zip serverpilot-tools-windows-386
	cd build && zip -r release/serverpilot-tools-$(VERSION)-Windows_arm64.zip serverpilot-tools-windows-arm64
	cd build/release && sha256sum serverpilot-tools-$(VERSION)-Linux_x86_64.tar.gz >> checksums.txt
	cd build/release && sha256sum serverpilot-tools-$(VERSION)-Linux_i386.tar.gz >> checksums.txt
	cd build/release && sha256sum serverpilot-tools-$(VERSION)-Linux_arm64.tar.gz >> checksums.txt
	cd build/release && sha256sum serverpilot-tools-$(VERSION)-Darwin_x86_64.tar.gz >> checksums.txt
	cd build/release && sha256sum serverpilot-tools-$(VERSION)-Darwin_arm64.tar.gz >> checksums.txt
	cd build/release && sha256sum serverpilot-tools-$(VERSION)-Windows_x86_64.zip >> checksums.txt
	cd build/release && sha256sum serverpilot-tools-$(VERSION)-Windows_i386.zip >> checksums.txt
	cd build/release && sha256sum serverpilot-tools-$(VERSION)-Windows_arm64.zip >> checksums.txt

.PHONY: clean
clean:
	rm -rf build
