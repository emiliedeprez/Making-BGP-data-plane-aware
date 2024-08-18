#!/bin/sh


# exabgp ${CONFIG}
while true
do
    mv /app/${PREFIXES} /app/prefixes.txt
    tcpdump -i any tcp port 179 -w /dump/provider1.pcap &
    exabgp ${CONFIG}
    sleep 1m
done