This folder contains the code for the second part of the master thesis. In the folder `BGP_server`, we find four folders and two files: 
- `config_measurement/` contains all configuration files used during measurements for routers running goBGP.
- `gobgp` is a copy of the gobgp repository at the commit 8fdda5dd2d6442c7c03636831bc536b67daa8f65 from OSRG https://github.com/osrg/gobgp with the modifications to add our solution.
- `goRQM` contains the code for the RQM process that we have developed and instruction to perform measurement of t-test and ranking.
- `host` contains the Dockerfile to build host used during our experiment.
- `Dockerfile` and `entrypoint` are two files used to build the docker used for our experiment.

In the folder `rpki_server`, we find one folder and multiples files:
- `gortr` is a copy of the gortr repository at the commit 05b892a754e6fb93763705c85424dc7a53c5b774 from Cloudflare https://github.com/cloudflare/gortr with the modifications to add our solution.
- `x_prefixes.txt` are files with 100, 1000, and 10000 prefixes generated to advertise
- `rpki_cache_x.json` are files with the modified ROAs used for our experiment.
- `create_rpki.py` is a python file that generates the prefixes and ROAs associated with for experiment.
- `Dockerfile` and `entrypoint` are two files used to build the docker used for our experiment.

In the folder `measurement`, we find one folder that contains Docker instructions and files to create the two providers running above exaBGP used in the experiment and 3 topologies files to deploy in containerlab the topologies used in our experiment.


To perform measurement

1. Go to BGP_server folder
`cd ./BGP_server`

2. Build the docker for bgp with rqm
`docker build . -t gobgp_rqm:1.0`

3. Go to rpki_server folder
`cd ../rpki_server`

4. Build the docker with the modified gortr and cache
`docker build . -t gortr:1.0`

5. Go to the provider1 folder
`cd ../measurement/measurement_docker/provider_1`

6. Build the docker
`docker build . -t provider1:latest`

7. Go to the provider2 folder
`cd ../provider_2`

8. Build the docker
`docker build . -t provider2:latest`

9. Return to the main folder
`cd ../..`

10. To run the experiment without the solution
`sudo containerlab deploy --topo topology_measurement_100.yml`

to stop : `sudo containerlab destroy --topo topology_measurement_100.yml`

