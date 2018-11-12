GO111MODULE := on

.PHONY: all
all:
	$(MAKE) fetch
	env GO111MODULE=${GO111MODULE} go run *.go

.PHONY: fetch
fetch:
	env GO111MODULE=${GO111MODULE} go get -v ./...
