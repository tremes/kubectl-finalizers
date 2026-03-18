NAME     := finalizers
PACKAGE  := github.com/tremes/$(NAME)
VERSION  := v0.0.1
GIT      := $(shell git rev-parse --short HEAD)
DATE     := $(shell date +%FT%T%Z)

build:     ## Builds the CLI
	go build \
	-ldflags "-w -X ${PACKAGE}/cmd.Version=${VERSION} -X ${PACKAGE}/cmd.Commit=${GIT} -X ${PACKAGE}/cmd.Date=${DATE}" \
    -a -o bin/${NAME} ./main.go

test-unit:
	go test -race -v ./...



