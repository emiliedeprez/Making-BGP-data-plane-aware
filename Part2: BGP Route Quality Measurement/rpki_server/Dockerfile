FROM golang:alpine3.19 as gortr_builder
WORKDIR /app
COPY gortr/go.mod gortr/go.mod
WORKDIR /app/gortr
RUN go mod download
WORKDIR /app
COPY gortr gortr
WORKDIR /app/gortr/cmd/gortr/
RUN go build -o gortr


FROM ubuntu:22.04
LABEL author="Emilie Deprez"
LABEL version="1.0"
WORKDIR /app
RUN apt-get update
RUN apt-get install python3 --yes
COPY entrypoint.sh .
COPY rpki_cache_100.json .
COPY rpki_cache_1000.json .
COPY rpki_cache_10000.json .
RUN chmod +x entrypoint.sh
COPY --from=gortr_builder /app/gortr/cmd/gortr .

ENV CONFIG="rpki_cache_100.json"

ENTRYPOINT ["./entrypoint.sh"]
