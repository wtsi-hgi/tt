export GOPATH := $(shell go env GOPATH)

# We require CGO_ENABLED=1 for getting group information to work properly; the
# pure go version doesn't work on all systems such as those using LDAP for
# groups
export CGO_ENABLED = 1

default: install

build:
	go build -tags netgo

install:
	@rm -f ${GOPATH}/bin/tt
	@go install -tags netgo
	@echo installed to ${GOPATH}/bin/tt

dev: export TT_ENV = development
dev:
	air server --url :4563 --cert cert.pem --key key.pem --logstderr

test:
	@go test -tags netgo --count 1 -p 1 ./...

race:
	go test -tags netgo -race --count 1 -p 1 ./...

bench:
	go test -tags netgo --count 1 -p 1 -run Bench -bench=. ./...

# curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.13.2
lint:
	@golangci-lint run --timeout 2m

clean:
	@rm -f ./tt
	@rm -f ./dist.zip

.PHONY: test race bench lint build install clean dev default