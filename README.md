# Gladius

> A [Mesos] framework with an API for running tests across a cluster.

## Running Gladius

Gladius can be run with [Docker]. By default, Gladius runs on port `8080`.

### Create ssh data volume container 
Keys in it, either using ssh-keygen or copying the keys into it. 
It's a long living container

```base
docker run -tid --volume /root/.ssh --name ssh ubuntu
````
 
###  Run Gladius server
```bash
docker run \
  --rm \
  --tty \
  --interactive \
  --volume $(which docker):$(which docker) \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  --publish 8080:8080 \
  typekit/gladius
```

### Ways to run Mesos locally

```bash
docker run \
  --name mesos \
  -d \
  -e MESOS_QUORUM=1 \
  -e MESOS_LOG_DIR=/var/log \
  -e MESOS_WORK_DIR=/tmp  \
  -p 5050:5050 redjack/mesos-master
```

```bash
docker run --privileged=true \
  -t -d --net="host" \
  -e MESOS_IP=192.168.59.103 \
  -e MESOS_LOG_DIR=/var/log \
  -e MESOS_MASTER=192.168.59.103:5050 \
  -e MESOS_CONTAINERIZERS=docker,mesos \
  -p 5051:5051 \
  -v $(which docker):$(which docker) \
  -v /var/run/docker.sock:/var/run/docker.sock razic/mesos-slave
```

[Docker]: https://docker.com
[Mesos]: http://mesos.apache.org/
