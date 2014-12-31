# Gladius

> A [Mesos] framework with an HTTP API for running tests across a cluster.

## Running Gladius

Gladius can be run with [Docker]. By default, Gladius runs on port `8080`.

### Create an SSH Key Container

This container will be used to authenticate with https://git.corp.adobe.com.

```bash
docker run \
  --tty \
  --interactive \
  --volume /root/.ssh \
  --name ssh \
  ubuntu
```

*Note:* Ensure the container has a passwordless RSA key in the `/root/.ssh`
directory, then add the corresponding public key to your
https://git.corp.adobe.com account.

### Start Mesos Master

```bash
docker run \
  --name mesos \
  --detach \
  --net host \
  --env MESOS_QUORUM=1 \
  --env MESOS_LOG_DIR=/var/log \
  --env MESOS_WORK_DIR=/tmp \
  --env MESOS_IP=192.168.59.103 \
  --env MESOS_PORT=5050 \
  --publish 5050:5050 \
  redjack/mesos-master
```

### Start Gladius

From the root of this directory:

```bash
bin/run gladius --master 192.168.59.103:5050 --executor /go/bin/gladius-executor --logtostderr
```

### Start Gladius Executor

See https://git.corp.adobe.com/typekit/gladius-executor

[Docker]: https://docker.com
[Mesos]: http://mesos.apache.org/
