.PHONY: vendor build build-all build-for-windows build-for-linux build-for-mac build-for-windows-amd64 build-for-windows-arm64 build-for-linux-amd64 build-for-linux-arm64 build-for-mac-amd64 build-for-mac-arm64

OUTPUT=output
APP=driver-box
BUILD=go build -o

vendor:
	rm -rf $(OUTPUT)/
	go mod tidy
	go mod vendor

build: vendor build-for-windows build-for-linux build-for-mac

build-for-windows: build-for-windows-amd64 build-for-windows-arm64

build-for-linux: build-for-linux-amd64 build-for-linux-arm64

build-for-mac: build-for-mac-amd64 build-for-mac-arm64

build-for-windows-amd64:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(BUILD) $(OUTPUT)/$(APP)-windows-amd64.exe main.go

build-for-windows-arm64:
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 $(BUILD) $(OUTPUT)/$(APP)-windows-arm64.exe main.go

build-for-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(BUILD) $(OUTPUT)/$(APP)-linux-amd64 main.go

build-for-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(BUILD) $(OUTPUT)/$(APP)-linux-arm64 main.go

build-for-mac-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(BUILD) $(OUTPUT)/$(APP)-mac-amd64 main.go

build-for-mac-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(BUILD) $(OUTPUT)/$(APP)-mac-arm64 main.go


