FROM python:3.9-alpine
LABEL author="Emilie Deprez"
LABEL version="1.0"
WORKDIR /app
RUN apk add tcpdump
RUN apk add iproute2-tc
RUN pip install exabgp
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh
COPY announce.py announce.py
COPY 100_prefixes.txt 100_prefixes.txt
COPY 1000_prefixes.txt 1000_prefixes.txt
COPY 10000_prefixes.txt 10000_prefixes.txt
COPY config_provider_1.conf config_provider_1.conf 
RUN exabgp --fi > /usr/local/etc/exabgp/exabgp.env
RUN mkfifo /run/exabgp.in
RUN mkfifo /run/exabgp.out
RUN chmod 777 /run/exabgp.in
RUN chmod 777 /run/exabgp.out
RUN chmod 777 /run
RUN mkdir /dump

ENV CONFIG="config_provider_1.conf"
ENV PREFIXES="100_prefixes.txt"

ENTRYPOINT ["./entrypoint.sh"]