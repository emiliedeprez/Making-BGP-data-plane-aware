#!/bin/sh

/etc/init.d/frr start

sleep 10s

while :
do
  # loop infinitely
  sleep 1
done

