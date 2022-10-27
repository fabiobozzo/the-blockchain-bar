GIT_COMMIT:=$(shell git rev-list -1 HEAD)
LDFLAGS:=-X main.GitCommit=${GIT_COMMIT}

install:
	go install -ldflags "$(LDFLAGS)" ./...

test:
	go test -v -p=1 -timeout=0 ./...