FROM golang:1.18.2-alpine3.14 as build
WORKDIR /go/src
COPY . .
RUN go build -o /app

FROM alpine:3.14
WORKDIR /usr/src
RUN addgroup -S gouser && adduser -S gouser -G gouser
USER gouser:gouser
COPY --from=build /app app

EXPOSE 8081
CMD ["./app"]

