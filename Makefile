.PHONY: build test clean docker unittest lint

ARCH=$(shell uname -m)
GO18=/usr/local/go/1.18/go/bin/go
GO=CGO_ENABLED=0 GO111MODULE=on ${GO18}
GOCGO=CGO_ENABLED=1 GO111MODULE=on ${GO18}


# see https://shibumi.dev/posts/hardening-executables
CGO_CPPFLAGS="-D_FORTIFY_SOURCE=2"
CGO_CFLAGS="-O2 -pipe -fno-plt"
CGO_CXXFLAGS="-O2 -pipe -fno-plt"
CGO_LDFLAGS="-Wl,-O1,–sort-common,–as-needed,-z,relro,-z,now"

MICROSERVICES=cmd/device-simple/device-simple
.PHONY: $(MICROSERVICES)

VERSION=0.0.0
DOCKER_TAG=$(VERSION)-dev

GOFLAGS=-ldflags "-X github.com/edgexfoundry/device-simple.Version=$(VERSION)"
CGOFLAGS=-ldflags "-linkmode=external -X github.com/edgexfoundry/device-sdk-go/v2.Version=$(VERSION)" -trimpath -mod=readonly -buildmode=pie
GOTESTFLAGS?=-race

GIT_SHA=$(shell git rev-parse HEAD)

tidy:
	go mod tidy -compat=1.17

build:
	$(GOCGO) build $(CGOFLAGS) -o $@ ./
	$(GOCGO) install -tags=safe

cmd/device-simple/device-simple:
	$(GOCGO) build $(CGOFLAGS) -o $@ ./example/cmd/device-simple

docker: vendor
	docker buildx build \
		-t driver-box:${VERSION} \
		--platform=linux/amd64,linux/arm64 \
		. --load

buildx:
	docker buildx rm builder
	docker buildx create --use --name builder
	docker buildx inspect builder --bootstrap

unittest:
	GO111MODULE=on go test $(GOTESTFLAGS) -coverprofile=coverage.out ./...

lint:
	@which golangci-lint >/dev/null || echo "WARNING: go linter not installed. To install, run\n  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.42.1"
	@if [ "z${ARCH}" = "zx86_64" ] && which golangci-lint >/dev/null ; then golangci-lint run --config .golangci.yml ; else echo "WARNING: Linting skipped (not on x86_64 or linter not installed)"; fi

test: unittest lint
	GO111MODULE=on go vet ./...
	gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")
	[ "`gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]
	./bin/test-attribution-txt.sh

clean:
	rm -f $(MICROSERVICES)

vendor:
	$(GO18) env -w GO111MODULE=on
	$(GO18) env -w GOPROXY=https://goproxy.cn,direct

	$(GO) mod tidy
	$(GO18) mod vendor
