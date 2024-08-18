#!/usr/bin/env python3

from __future__ import print_function

from sys import stdout
from time import sleep
from threading import Thread

sleep(80)

def stdin_reader():
    stdin.readlines()

t = Thread(target=stdin_reader)
t.start()

f = open("/app/prefixes.txt", "r")
for prefix in f.readlines():
    stdout.write("announce route " + prefix.strip() + " next-hop self as-path [65003 65007 65006 65005]\n")
    stdout.flush()
    sleep(0.1)
f.close()

while True:
    sleep(1)