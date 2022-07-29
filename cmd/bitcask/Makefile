.DEFAULT_GOAL := dev
.PHONY: clean

dev:
	@go build -ldflags="-X main.GitCommit=$(shell git rev-parse HEAD)"

mac:
	@GOOS="darwin" GOARCH="amd64" go build -ldflags="-X main.GitCommit=$(shell git rev-parse HEAD)"

linux:
	@GOOS="linux" GOARCH="amd64" go build -ldflags="-X main.GitCommit=$(shell git rev-parse HEAD)"

clean:
	@rm -f ./bitcask