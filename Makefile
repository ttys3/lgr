ci/test:
	go test -v ./...

test:
	command -v richgo || (WORK=$(shell pwd) && cd /tmp && GO111MODULE=on go get -u github.com/kyoh86/richgo && cd $(WORK))
	richgo test -v ./...

fmt:
	command -v gofumpt || (WORK=$(shell pwd) && cd /tmp && GO111MODULE=on go get mvdan.cc/gofumpt && cd $(WORK))
	gofumpt -w -s -d .

lint:
	golangci-lint run  -v

ci/lint: export GOPATH=/go
ci/lint: export GO111MODULE=on
ci/lint: export GOPROXY=https://goproxy.cn
ci/lint: export GOOS=linux
ci/lint: export CGO_ENABLED=0
ci/lint: lint
