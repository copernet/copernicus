GOPATH ?= $(shell go env GOPATH)

# Ensure GOPATH is set before running build process.
ifeq "$(GOPATH)" ""
  $(error Please set the environment variable GOPATH before running `make`)
endif

CURDIR := $(shell pwd)
path_to_add := $(addsuffix /bin,$(subst :,/bin:,$(GOPATH)))
export PATH := $(path_to_add):$(PATH)

GO        := go
GOBUILD   := CGO_ENABLED=0 $(GO) build $(BUILD_FLAG)
GOTEST    := CGO_ENABLED=1 $(GO) test -p 3
OVERALLS  := CGO_ENABLED=1 overalls
GOVERALLS := goveralls

ARCH      := "`uname -s`"
LINUX     := "Linux"
MAC       := "Darwin"
PACKAGES  := $$(go list ./...| grep -vE "vendor")
FILES     := $$(find . -name "*.go" | grep -vE "vendor")
TOPDIRS   := $$(ls -d */ | grep -vE "vendor")

GOFAIL_ENABLE  := $$(find $$PWD/ -type d | grep -vE "(\.git|vendor)" | xargs gofail enable)
GOFAIL_DISABLE := $$(find $$PWD/ -type d | grep -vE "(\.git|vendor)" | xargs gofail disable)

.PHONY: all build update test

RACE_FLAG =
ifeq ("$(WITH_RACE)", "1")
	RACE_FLAG = -race
	GOBUILD   = GOPATH=$(GOPATH) CGO_ENABLED=1 $(GO) build
endif

all:

build:
    $(GOBUILD)

update:
    which glide >/dev/null || curl https://glide.sh/get | sh
    which glide-vc || go get -v -u github.com/sgotti/glide-vc
    rm -r vendor && mv _vendor/src vendor || true
    rm -rf _vendor
ifdef PKG
    glide get -s -v --skip-test ${PKG}
else
    glide update -s -v -u --skip-test
endif
    @echo "removing test files"
    glide vc --only-code --no-tests
    mkdir -p _vendor
    mv vendor _vendor/src

test: gotest

clean:
	$(GO) clean -i ./...
	rm -rf *.out

fmt:
	@echo "gofmt (simplify)"
	@ gofmt -s -l -w $(FILES) 2>&1 | grep -v "vendor|parser/parser.go" | awk '{print} END{if(NR>0) {exit 1}}'

gofail-enable:
# Converting gofail failpoints...
	@$(GOFAIL_ENABLE)

gofail-disable:
# Restoring gofail failpoints...
	@$(GOFAIL_DISABLE)