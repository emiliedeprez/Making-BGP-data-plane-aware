import ipaddress

base_network = int(ipaddress.ip_address("10.0.0.0"))

f = open("10000_prefixes.txt", "w")
for i in range(1, 1001):
    f.write(str(ipaddress.ip_network(str(ipaddress.ip_address(base_network))+"/24")) + "\n")
    base_network += 65536

f.close()