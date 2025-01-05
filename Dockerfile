FROM golang:1.23.4 AS build
RUN apt-get update && apt-get install -y libvips && apt-get install -y libvips-dev && apt -y install chromium
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . /app
WORKDIR /app/cmd/api
RUN go build -o /app/bin/sad-app
WORKDIR /app
ENTRYPOINT [ "bin/sad-app" ]
