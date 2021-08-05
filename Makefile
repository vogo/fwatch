version := v1.0.0

format:
		goimports -w -l .
		go fmt
		gofumpt -w .

license-check:
	license-header-checker -v -a -r apache-license.txt . go

check: license-check
		golangci-lint run

test:
		go test -coverprofile=coverage.txt -covermode=atomic

test-linux:
		docker run --rm -v $(shell pwd):/projectdir \
        -v ${GOPATH}/src:/go/src \
        -v ${GOPATH}/pkg:/go/pkg \
        -w /projectdir \
        -e GOOS="linux" \
        -e GOARCH="amd64" \
        -e GOPROXY=https://goproxy.io \
        golang:1.16-buster \
        go test -v

build: format check test

package:
	mkdir -p dist
	rm -f dist/*.zip
	cd dist && GOOS=linux go build ../cmd/fwatch/fwatch.go && zip fwatch-$(version)-linux.zip fwatch && rm -f fwatch



