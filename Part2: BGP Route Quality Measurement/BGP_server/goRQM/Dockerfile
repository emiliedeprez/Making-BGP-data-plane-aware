FROM golang:alpine3.19 
WORKDIR /app
COPY . .
RUN go build -o main

ENTRYPOINT ./main --action measurement

