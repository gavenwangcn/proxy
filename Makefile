ROOT_DIR 		= $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))/
VERSION_PATH	= $(shell echo $(ROOT_DIR) | sed -e "s;${GOPATH}/src/;;g")pkg/util
LD_GIT_COMMIT   = -X '$(VERSION_PATH).GitCommit=`git rev-parse --short HEAD`'
LD_BUILD_TIME   = -X '$(VERSION_PATH).BuildTime=`date +%FT%T%z`'
LD_GO_VERSION   = -X '$(VERSION_PATH).GoVersion=`go version`'
LD_FLAGS        = -ldflags "$(LD_BUILD_TIME) $(LD_GO_VERSION) -w -s"

GOOS 		= linux
CGO_ENABLED = 0
DIST_DIR 	= $(ROOT_DIR)dist/

DOCKER_TAG			:= $(tag)

ifeq ("$(DOCKER_TAG)","")
	DOCKER_TAG		:= $(shell date +%Y%m%d%H%M)
endif

.PHONY: release
release: dist_dir proxy;

.PHONY: release_darwin
release_darwin: darwin dist_dir proxy;

.PHONY: docker
docker: release ;
	@echo ========== current docker tag is: $(DOCKER_TAG) ==========
	docker build -t dylanwangcn/proxy:$(DOCKER_TAG) -f Dockerfile-proxy .

.PHONY: darwin
darwin:
	$(eval GOOS := darwin)

.PHONY: proxy
proxy: ; $(info ======== compiled proxy:)
	env CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) go build -a -installsuffix cgo -o $(DIST_DIR)proxy $(LD_FLAGS) $(ROOT_DIR)cmd/proxy/*.go
	cp $(ROOT_DIR)cmd/proxy/*.json $(DIST_DIR)
	cp $(ROOT_DIR)entrypoint.sh $(DIST_DIR)
.PHONY: dist_dir
dist_dir: ; $(info ======== prepare distribute dir:)
	mkdir -p $(DIST_DIR)
	@rm -rf $(DIST_DIR)*

.PHONY: clean
clean: ; $(info ======== clean all:)
	rm -rf $(DIST_DIR)*

.PHONY: help
help:
	@echo "build release binary: \n\t\tmake release\n"
	@echo "build Mac OS X release binary: \n\t\tmake release_darwin\n"
	@echo "build docker release : \n\t\tmake docker\n"
	@echo "clean all binary: \n\t\tmake clean\n"

UNAME_S := $(shell uname -s)

# 设置默认编译目标
ifeq ($(UNAME_S),Darwin)
	.DEFAULT_GOAL := release_darwin
else
	.DEFAULT_GOAL := release
endif
