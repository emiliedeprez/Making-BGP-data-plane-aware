#!/bin/sh

while true
do
    python3 -m http.server 8000 &
    sleep 5s
    ./gortr --cache http://localhost:8000/${CONFIG} --verify=false --checktime=false -bind 172.20.20.17:8323
    sleep 1m
done