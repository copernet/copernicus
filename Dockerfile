FROM golang

WORKDIR /go/src/app

RUN curl https://glide.sh/get | sh