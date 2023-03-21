## Build
FROM golang:1.20-buster AS build

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

WORKDIR cmd/pubsub/publisher/
ENV CGO_ENABLED=0
RUN go build  -a -installsuffix cgo -o publisher .
## Deploy
FROM alpine
COPY --from=build /app/cmd/pubsub/publisher/ ./
EXPOSE 8080
ENTRYPOINT ["./publisher"]