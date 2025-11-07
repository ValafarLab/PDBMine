FROM golang:1.16 AS builder
WORKDIR /pdbmine

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY *.go ./
# Edit directory of Database.dat created from pdbmine_db if necessary
COPY Database.dat .

RUN CGO_ENABLED=0 GOOS=linux go build -o DiReDB .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /pdbmine/DiReDB .
COPY --from=builder /pdbmine/Database.dat .

EXPOSE 8077
CMD ["./DiReDB"]
