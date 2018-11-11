GO111MODULE := on

.PHONY: all
all:
	env GO111MODULE=${GO111MODULE} go run *.go

