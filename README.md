# Gladius

> A [Mesos] framework with an API for running tests across a cluster.

## Running Gladius

Gladius can be run with [Docker]. By default, Gladius runs on port `8080`.

```bash
docker run \
  --rm \
  --tty \
  --interactive \
  --volume $(which docker):$(which docker) \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  typekit/gladius
```

[Docker]: https://docker.com
[Mesos]: http://mesos.apache.org/
