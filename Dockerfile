FROM golang:latest

WORKDIR /app
ADD . /app

RUN go build

CMD ./rssh --loglevel=debug server