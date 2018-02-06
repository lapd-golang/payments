FROM golang:1.9.3

RUN go get -u github.com/rampage644/payments || true
RUN go get -u github.com/golang/dep/cmd/dep
WORKDIR /go/src/github.com/rampage644/payments
RUN dep ensure


RUN go install github.com/rampage644/payments/service
EXPOSE 8080
CMD ["/go/bin/service"]
