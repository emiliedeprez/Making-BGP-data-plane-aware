This folder contains the code for the first part of the master thesis. In the folder `base_gobgp`, we find two folders and two files:
- `gobgp` is a copy of the gobgp repository at the commit 8fdda5dd2d6442c7c03636831bc536b67daa8f65 from OSRG https://github.com/osrg/gobgp used as base for our experiment.
- `measurement_config/` contains all configuration files used during measurements for routers running goBGP.
- `Dockerfile` and `entrypoint` are two files used to build the docker used for our experiment.

In the folder `planned-shutdown_gobgp`, we find two folders and two files and two files:
- `gobgp` is a copy of the gobgp repository at the commit 8fdda5dd2d6442c7c03636831bc536b67daa8f65 from OSRG https://github.com/osrg/gobgp with our solution.
- `measurement_config/` contains all configuration files used during measurements for routers running goBGP.
- `Dockerfile` and `entrypoint` are two files used to build the docker used for our experiment.

In the folder `measurement`, we find one folder that contains Docker instructions and files to create the two providers running above exaBGP used in the experiment and 2 topologies files to deploy in containerlab the topologies used in our experiment.


To perform measurement

1. Go to base_gobgp folder
`cd ./base_gobgp`

2. Build the base docker
`docker build . -t gobgp:1.0`

3. Go to local-pref_bgp folder
`cd ../planned-shutdown_bgp`

4. Build the docker with the solution
`docker build . -t gobgp:2.0`

5. Go to the provider1 folder
`cd ../measurement/measurement_docker/provider_1`

6. Build the docker
`docker build . -t provider_1:latest`

7. Go to the provider2 folder
`cd ../provider_2`

8. Build the docker
`docker build . -t provider_2:latest`

9. Return to the main folder
`cd ../..`

10. To run the experiment without the solution
`sudo containerlab deploy --topo measurement_base.yml`

to stop : `sudo containerlab destroy --topo measurement_base.yml`

11. To run the experiment with the solution
`sudo containerlab deploy --topo measurement_planned-shutdown.yml`

to stop: `sudo containerlab destroy --topo measurement_planned-shutdown.yml`
