# PCBOOK

This is meant to be an exploration of gRPC.

### Setup
``` shell
# Start docker

# build container
docker build -t <DOCKERID>/pc-book .

# run container
docker run -it <DOCKERID>/pc-book

#
# NOTE: All following commands happen in a second terminal tab
#

# check container is running, copy container id
docker ps

# enter container
docker exec -it <CONTAINER ID> sh

# run client or test
# option 1
make client
# option 2
make test
```

