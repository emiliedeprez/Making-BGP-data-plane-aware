#!/bin/sh

/etc/init.d/frr start

sleep 10s


# while :
# do
#   # loop infinitely
# done

./main -config "config_measurement/rqm.yaml" &
./gobgpd -f ${CONFIG}
sleep 10s

