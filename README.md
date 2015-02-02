# Gladius

> A [Mesos] framework with an HTTP API for running tests across a cluster.

## Usage

*Important:* Ensure the Gladius container has a passwordless RSA key in the
`/root/.ssh` directory, then add the corresponding public key to your
https://git.corp.adobe.com account. Also, Gladius needs to connect to a Redis
database for persistence.

### Development

```bash
docker run \
  --detach \
  --privileged \
  --net host \
  --name gladius \
  --env GLADIUS_HTTP_PORT=8080 \
  --env REDIS_TCP_ADDR=:6379 \
  --env MESOS_MASTER=192.168.27.3:80 \
  --env EXEC_URI=/executors/gladius \
  --volume $(which docker):$(which docker) \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  docker.corp.adobe.com/typekit/gladius
```

### Production

```bash
docker run \
  --detach \
  --privileged \
  --net host \
  --name gladius \
  --env GLADIUS_HTTP_PORT=8080 \
  --env REDIS_TCP_ADDR=:6379 \
  --env MESOS_MASTER=192.168.27.3:80 \
  --env EXEC_URI=/executors/gladius \
  --volume $(which docker):$(which docker) \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  docker.corp.adobe.com/typekit/gladius
```

[Mesos]: http://mesos.apache.org/
