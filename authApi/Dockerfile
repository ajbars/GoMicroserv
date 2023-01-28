FROM golang:1.18.2-alpine3.14 as build
WORKDIR /go/src
COPY . .
RUN go build -o /app

FROM alpine:3.14
RUN apk update && \
    apk add --no-cache \
    openssh-keygen
WORKDIR /usr/src
RUN ssh-keygen -q -t ecdsa -m PEM -N '' -f ./id_ecdsa
RUN addgroup -S gouser && adduser -S gouser -G gouser
RUN chown gouser ./id_ecdsa ./id_ecdsa.pub
USER gouser:gouser
COPY --from=build /app app

EXPOSE 8082
CMD ["./app"]

