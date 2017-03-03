# This dockerfile builds a monoimage that contains
# all microservices. Use ENTRYPOINT/CMD to specify
# which microservice you want to execute.
FROM golang:1.8-alpine

RUN apk add --update git && \
	rm -rf /var/cache/apk/*

COPY . /go/src/github.com/ahmetb/coffeelog
WORKDIR src/github.com/ahmetb/coffeelog
RUN go install -v -ldflags "-X github.com/ahmetb/coffeelog/version.version=$(git describe --always --dirty)" ./... 2>&1

RUN git status > /status.txt
RUN git diff > /diff.txt

# 'web' service requires static files and templates
# to be present within ./static
# TODO read static files dir from env/cfg
WORKDIR web

# TODO make PORT configurable in each program so we
# do not need to list all ports here
EXPOSE 8000 8001 8002

# CMD/ENTRYPOINT is not listed here, it's in pod template.
