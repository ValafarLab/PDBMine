# PDB Mine
PDBMine is a solution for fast queries across all PDB's in the Protein Data Bank owned and maintained at https://www.rcsb.org. This application is a series of Python scripts that work together to download all of the PDB files and then generate a DSSP (https://swift.cmbi.umcn.nl/gv/dssp/) file per model in the PDB file. The data in the DSSP file is then combined into one "database" file that supports queries across all PDB's using set arithmetic.

## Python Versions and Packages


## DSSP


## Config file

### Rsync

### Results


## Docker
Docker is utilized to reduce dependancy issues and increase portability across environments. The Docker file defines a Docker image that includes all of the scripts along with any required software. This allows us to then deploy the container on any system that supports Docker.

### Building the container
The container can be built using the following command:

`docker build --rm -t <uid>/pdbmine_build .`

### Running the container
The container can be run using the following command:

`docker run -it --mount type=bind,source="/Volumes/ProteinData",target=/data/ProteinData <uid>/pdbmine_build`

Note that an external volume should be mounted so that the PDB's can be saved as their downloaded instead of pulling them every time.