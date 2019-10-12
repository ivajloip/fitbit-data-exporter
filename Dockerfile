# Copyright (C) 2019 All Rights Reserved
# Author: Ivaylo Petrov ivajloip@gmail.com

FROM golang:1.13-alpine as dev

# install tools
RUN apk add --no-cache git make upx

ARG PROJECT_NAME=fitbit-data-exporter

ARG BASE_PATH=github.com/ivajloip/

ARG VERSION

ENV PROJECT_PATH=/go/src/$BASE_PATH/$PROJECT_NAME

# setup work directory
RUN mkdir -p $PROJECT_PATH
WORKDIR $PROJECT_PATH

COPY go.mod go.sum $PROJECT_PATH/

RUN go mod download

# copy source code
COPY . $PROJECT_PATH

ARG FAST_BUILD=false

# build
RUN VERSION=${VERSION} make build && ( [ "$FAST_BUILD" = "true" ] || upx -9 $PROJECT_PATH/build/$PROJECT_NAME )

# runnable
FROM alpine:3.8 as runnable

RUN addgroup -S app && adduser -S -G app app \
      && mkdir /home/app/.config && chown app:app /home/app/.config

ENTRYPOINT ["/bin/fitbit-data-exporter", "api"]

ARG PROJECT_NAME=fitbit-data-exporter

ARG BASE_PATH=github.com/ivajloip/

COPY --from=dev /go/src/$BASE_PATH/$PROJECT_NAME/build/${PROJECT_NAME} /bin/${PROJECT_NAME}

USER app
