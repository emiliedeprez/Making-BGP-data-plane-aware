import ipaddress
import random
import json 

base_network = int(ipaddress.ip_address("10.0.0.0"))
all_prefix = []

nbr_prefix = 10000
f = open(f"{nbr_prefix}_prefixes.txt", "w")
for i in range(1, nbr_prefix+1):
    f.write(str(ipaddress.ip_network(str(ipaddress.ip_address(base_network))+"/24")) + "\n")
    all_prefix.append(str(ipaddress.ip_network(str(ipaddress.ip_address(base_network))+"/24")))
    base_network += 65536
f.close()

random.shuffle(all_prefix)

ROAS = {"roas": []}

for i in range(int(nbr_prefix/100)*35):
    ROAS["roas"].append({
        "prefix": all_prefix[i],
        "asn": "AS65005",
        "measurementIP": [str(ipaddress.IPv4Network('110.0.0.0/16')[i])]
    })

for i in range(int(nbr_prefix/100)*35):
    ROAS["roas"].append({
        "prefix": all_prefix[int(nbr_prefix/100)*35+i],
        "asn": "AS65005",
        "measurementIP": [str(ipaddress.IPv4Network('130.0.0.0/16')[i])]
    })

for i in range(int(nbr_prefix/100)*30):
    ROAS["roas"].append({
        "prefix": all_prefix[int(nbr_prefix/100)*70+i],
        "asn": "AS65005",
        "measurementIP": [str(ipaddress.IPv4Network('120.0.0.0/16')[i])]
    })


f = open(f"rpki_cache_{nbr_prefix}.json", "w")
json.dump(ROAS, f)
f.close()