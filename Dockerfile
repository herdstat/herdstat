#
# Copyright (c) 2023 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
#
# SPDX-License-Identifier: MIT
#

# syntax=docker/dockerfile:1

# Start by building the application.
FROM golang:1.19 as build

# Build variable to control environment. Set to either "debug" or leave undefined.
ARG ENV

# Build Delve
RUN if [ "$ENV" = "debug" ]; then go install github.com/go-delve/delve/cmd/dlv@v1.20.1; fi

# Build libmergestat.so
RUN apt-get update && apt-get -y install cmake libssl-dev
RUN git clone --recurse-submodules https://github.com/mergestat/mergestat-lite.git
WORKDIR mergestat-lite
RUN git checkout v0.5.10
RUN make libgit2
RUN make .build/libmergestat.so

COPY cmd /app/cmd/
COPY internal /app/internal/
COPY go.mod go.sum /app/
COPY main.go /app/
WORKDIR /app

# compile application
RUN go build

# Now copy it into our base image.
FROM gcr.io/distroless/base-debian11

COPY --from=build /go/mergestat-lite/.build/libmergestat.so /usr/lib/
COPY --from=build /app/herdstat /go/bin/dlv* /

