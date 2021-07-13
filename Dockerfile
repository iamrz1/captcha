FROM golang:1.14-alpine AS builder
ENV GO111MODULE=on
ENV PORT=8000

RUN mkdir /tmpdir

WORKDIR  /tmpdir
COPY go.mod .
# COPY go.sum .
RUN go mod download

COPY . . 

RUN go build -o server
# Defining App image


FROM alpine:latest
RUN apk add --no-cache --update ca-certificates

WORKDIR /app
# Copy App binary to image
COPY --from=builder /tmpdir/server /app/
RUN touch .env
EXPOSE 8000

ENTRYPOINT ["./server"]
