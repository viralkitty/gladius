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

### 
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

[Docker]: https://docker.com
[Mesos]: http://mesos.apache.org/
