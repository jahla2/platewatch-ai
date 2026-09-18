FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/platewatch-server ./cmd/api

FROM alpine:3.21
COPY --from=build /out/platewatch-server /usr/local/bin/platewatch-server
EXPOSE 8080
ENTRYPOINT ["platewatch-server"]
