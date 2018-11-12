GO111MODULE := on
CAN := $(shell echo "$(go version)" | grep -qE "1.11|1.12" && echo 1 || echo 0)

.PHONY: all
all:
ifeq ($(CAN), 0)
	$(error Must have Go 1.11 or higher)
endif
	@$(MAKE) fetch
	@env GO111MODULE=${GO111MODULE} go run *.go

.PHONY: fetch
fetch:
	env GO111MODULE=${GO111MODULE} go get -v ./...
