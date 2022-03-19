# ShoppiDB
A highly available, reliable, and incrementally scalable distributed system!

## Docker (How to run it)
```
#Start the containers
docker-compose up -d --build --remove-orphans <service name>
```
Helpful commands:
```
#To view container's details :
docker ps
#To view stdout:
docker-compose logs -f <service name>
#To run commands in container:
docker exec <container id> <commands>
# To get interactive shell for container
docker exec -it <container id> /bin/bash
```