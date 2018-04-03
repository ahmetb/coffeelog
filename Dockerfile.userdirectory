FROM golang:1.9-alpine as build
COPY . $GOPATH/src/github.com/ahmetb/coffeelog
ARG REVISION_ID
RUN go install -v \
      -tags netgo \
      -ldflags="-w -X github.com/ahmetb/coffeelog/version.version=$REVISION_ID" \
      github.com/ahmetb/coffeelog/cmd/userdirectory

FROM alpine
RUN apk add --update ca-certificates && \
      rm -rf /var/cache/apk/* /tmp/*

COPY  --from=0 /go/bin/userdirectory ./userdirectory
ENTRYPOINT ["./userdirectory"]
EXPOSE 8001
