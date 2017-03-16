BIN_DIR=./gopath/bin
BINARIES=web coffeedirectory userdirectory
PROJECT?=ahmetb-starter

docker-images: binaries
	BINS=(${BINARIES}); for b in $${BINS[*]}; do \
	  docker build -f=Dockerfile.$$b -t=gcr.io/${PROJECT}/$$b:latest . ;\
	done
binaries:
	if [ -z "$$GOPATH" ]; then echo "GOPATH is not set"; exit 1; fi
	@echo "Building statically compiled linux/amd64 binaries"
	set -x; BINARIES=(web userdirectory coffeedirectory); \
	  GOOS=linux GOARCH=amd64 go install \
	  -a -tags netgo \
	  -ldflags="-w -X github.com/ahmetb/coffeelog/version.version=$$(git describe --always --dirty)" \
	    $(patsubst %, ./%, $(BINARIES)) && \
	rm -rf ${BIN_DIR} && mkdir -p ${BIN_DIR} && \
	cp $(patsubst %, $$GOPATH/bin/linux_amd64/%, $(BINARIES)) ${BIN_DIR}

PHONY: .docker-images
