# PDBMine (database builder) + DiReDB (query service)

This repository contains two components:

- pdbmine_db: Python pipeline that downloads PDB structures, generates DSSP files, and compiles a compact binary database file.
- pdbmine_service: Go REST API that loads the database file into memory and serves query endpoints.

Typical flow: build the database once with pdbmine_db, then run pdbmine_service to query it.

## Prerequisites
- Docker (Linux/macOS/Windows). No local Python/Go setup required if you use the images.

## Quick start

### 1) Build the database

The builder image downloads PDB data and writes outputs into a mounted host folder.

```bash
cd pdbmine_db
# Build the image
docker build --rm -t yourname/pdbmine_build .

# Choose a host folder for outputs
mkdir -p /path/to/ProteinData

# Run the build (this takes time on first run), previous benchmark was 27 hours on an AWS t3.2xlarge instance (8vCPUs, 32 GB Ram)
# On Linux, replace /path/to/ProteinData with your absolute path.
docker run -it --rm \
  --mount type=bind,source="/path/to/ProteinData",target=/data/ProteinData \
  yourname/pdbmine_build
```

Outputs (with the default config in docker_config.yaml):
- PDB files: /data/ProteinData/pdb
- DSSP files: /data/ProteinData/dssp
- Database file: /data/ProteinData/db3_0_1.dat
- Logs: /data/ProteinData/log/pdbmine.log

Tips:
- Progress appears in container logs. You can also tail the host log file.
- You can limit work for a quick demo by setting `runtime.limit-proteins: true` and providing a small `protein-list` in config.yaml.

### 2) Run the API service

The service expects a database file. Two options:

A) Bake the DB into the image (simple for users)
```bash
# Copy or rename the generated DB to the service folder as Database.dat
cp /path/to/ProteinData/db3_0_1.dat pdbmine_service/Database.dat

cd pdbmine_service
# Build the API image (Dockerfile copies Database.dat into the image)
docker build -t diredb .

# Run the API
# Exposes port 8077. Database is already inside the image at /root/Database.dat
# Optionally override port with DIREDB_PORT
docker run --rm -it -p 8077:8077 diredb
```

B) Mount the DB at runtime (avoids baking large files)

Note: the current Dockerfile copies Database.dat at build time. To use runtime mounting, either place a small placeholder Database.dat when building or adjust the Dockerfile to skip copying it. Then run:
```bash
# Build service image (ensure Dockerfile doesn’t require Database.dat)
cd pdbmine_service
docker build -t diredb .

# Run with the DB mounted and path set via DIREDB_DB
docker run --rm -it -p 8077:8077 \
  -e DIREDB_DB=/data/Database.dat \
  -v /path/to/ProteinData/db3_0_1.dat:/data/Database.dat:ro \
  diredb
```

Environment variables supported by the service:
- DIREDB_PORT: port to listen on (default 8077)
- DIREDB_DB: absolute path to the database file inside the container (default: /root/Database.dat as baked by the Dockerfile)

Memory note: the service loads the DB into RAM (~6 GB for the full dataset). Ensure the host has sufficient memory.

### 3) Try an endpoint
```bash
curl http://localhost:8077/v1/api/database/header
```
See more endpoints in `pdbmine_service/README.md`.

## Repository structure
```
/ (this repo)
  README.md
  LICENSE
  pdbmine_db/
    Dockerfile, buildDB.py, config files, Python modules
  pdbmine_service/
    Dockerfile, Go sources, README.md
```

## Building locally (optional)
- Python builder requires: Python 3.8, dssp, rsync, and packages in `requirements.txt`.
- Go service builds with Go 1.16+.
Docker is recommended to avoid local setup.

## License
See LICENSE.

## Acknowledgements
- Protein Data Bank (RCSB) for source structures.
- DSSP for secondary structure assignment.
