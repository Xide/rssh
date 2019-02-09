# Builder stage
FROM golang:1.11-alpine AS deps_dl

RUN apk add --no-cache git

WORKDIR /rssh

COPY go.mod go.sum ./
RUN go mod download

FROM deps_dl as builder

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -o rssh ./

# Release stage
FROM scratch AS release

WORKDIR /bin

COPY --from=builder /rssh/rssh /bin/rssh

ENTRYPOINT [ "/bin/rssh" ]
