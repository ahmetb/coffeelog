FROM golang:1.8-alpine

COPY . /go/src/github.com/ahmetalpbalkan/coffeelog
WORKDIR src/github.com/ahmetalpbalkan/coffeelog
RUN go install -v ./...

# 'web' service requires static files and templates
# to be present within ./static
# TODO read static files dir from env/cfg
WORKDIR web

# TODO make PORT configurable in each program so we
# do not need to list all ports here
EXPOSE 8000 8001 8002

# CMD/ENTRYPOINT is not listed here, it's in pod template.
