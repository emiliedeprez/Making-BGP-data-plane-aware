import ipaddress

base_network = int(ipaddress.ip_address("10.0.0.0"))

nbr_prefix = 10000
f = open(f"{nbr_prefix}_prefixes.txt", "w")
for i in range(1, nbr_prefix+1):
    f.write(str(ipaddress.ip_network(str(ipaddress.ip_address(base_network))+"/24")) + "\n")
    base_network += 65536

f.close()