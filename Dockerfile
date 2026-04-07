ARG GO_VERSION=1.26.1

FROM golang:${GO_VERSION}-alpine

ENV GO111MODULE=on LANG=en_US.UTF-8

RUN mkdir -p $GOPATH/src

WORKDIR /src

COPY . .

RUN CGO_ENABLED=0 go build . \
    && mv merge-gatekeeper /go/bin/ \
    && addgroup -S mergegatekeeper \
    && adduser -S -G mergegatekeeper mergegatekeeper

USER mergegatekeeper

ENTRYPOINT ["/go/bin/merge-gatekeeper"]
