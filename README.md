# Gladius

> A [Mesos] framework with an API for running tests across a cluster.

## Running Gladius

Gladius can be run with [Docker]. By default, Gladius runs on port `8080`.

### Start [Mesos]

Use the link below to set up your [Mesos] cluster locally. You'll need to follow
all the steps except the Marathon part.

http://viralkitty.com/marathon-and-docker

### Create an SSH Key Container

This container is will be used to authenticate to https://git.corp.adobe.com.

```bash
docker run \
  --tty \
  --interactive \
  --volume /root/.ssh \
  --name ssh \
  ubuntu
```

```
docker run --name mesos -d --net="host" -e MESOS_QUORUM=1 -e MESOS_LOG_DIR=/var/log -e MESOS_WORK_DIR=/tmp -e MESOS_IP=192.168.59.103 -e MESOS_PORT=5050 -p 5050:5050 redjack/mesos-master
docker run --privileged=true -t -d --net="host" -e MESOS_IP=192.168.59.103 -e MESOS_LOG_DIR=/var/log -e MESOS_MASTER=192.168.59.103:5050 -e MESOS_CONTAINERIZERS=docker,mesos -p 5051:5051 -v $(which docker):$(which docker) -v /var/run/docker.sock:/var/run/docker.sock --volumes-from $(docker ps | grep gladius:latest | awk '{print $1}') razic/mesos-slave
```

*Note:* Ensure sure the container has a passwordless RSA key in the
`/root/.ssh` directory, then add the corresponding public key to your
https://git.corp.adobe.com account.

###  Run Gladius

From the root of this directory:

```bash
./run.sh gladius --master $MESOS_MASTER_HOST:$MESOS_MASTER_PORT --logtostderr
```

[Docker]: https://docker.com
[Mesos]: http://mesos.apache.org/
