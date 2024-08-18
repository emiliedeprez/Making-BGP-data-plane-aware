#!/usr/bin/env python3

from __future__ import print_function

from sys import stdin, stdout
from time import sleep
from threading import Thread


sleep(5)

def stdin_reader():
    stdin.readlines()

t = Thread(target=stdin_reader)
t.start()

#Iterate through messages
f = open("/app/prefixes.txt", "r")
for prefix in f.readlines():
    stdout.write("announce route " + prefix.strip() + " next-hop self as-path [65002 65007] community [65535:65285]\n")
    stdout.flush()
f.close()

sleep(120)

f = open("/app/prefixes.txt", "r")
for prefix in f.readlines():
    stdout.write("withdraw route " + prefix.strip() + "\n")
    stdout.flush()
f.close()

#Loop endlessly to allow ExaBGP to continue running
while True:
    sleep(1)