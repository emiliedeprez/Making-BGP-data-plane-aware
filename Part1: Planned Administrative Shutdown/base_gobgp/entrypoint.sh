#!/bin/sh


while true
do
    tcpdump -i any tcp port 179 -w /dump/${PCAP} &
    ./gobgpd -f ${CONFIG}
    sleep 1m
done