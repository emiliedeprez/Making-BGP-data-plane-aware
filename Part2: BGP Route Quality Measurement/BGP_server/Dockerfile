FROM golang:alpine3.19 AS gobgp_builder
WORKDIR /app
COPY gobgp/go.mod gobgp/go.mod
WORKDIR /app/gobgp
RUN go mod download
WORKDIR /app
COPY gobgp gobgp
WORKDIR /app/gobgp/cmd/gobgp
RUN go build -o gobgp
WORKDIR /app/gobgp/cmd/gobgpd
RUN go build -o gobgpd

FROM golang:alpine3.19 AS gorqm_builder
WORKDIR /app
WORKDIR /app
COPY goRQM goRQM
WORKDIR /app/goRQM
RUN go build -o main


FROM ubuntu:22.04
LABEL author="Emilie Deprez"
LABEL version="1.0"
WORKDIR /app
RUN apt-get update
RUN apt-get install tcpdump --yes
RUN apt-get install frr --yes
RUN apt-get install iputils-ping --yes
RUN apt-get install apt-utils --yes
RUN apt-get install kmod --yes
RUN apt-get install traceroute --yes
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh
COPY config_measurement config_measurement
COPY --from=gobgp_builder /app/gobgp/cmd/gobgp/gobgp /bin
COPY --from=gobgp_builder /app/gobgp/cmd/gobgpd/gobgpd .
COPY --from=gorqm_builder /app/goRQM/main .
RUN mkdir dump

ENV CONFIG="config/AS65001"

ENTRYPOINT ["./entrypoint.sh"]
