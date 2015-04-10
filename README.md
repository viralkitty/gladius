# Gladius

> A [Mesos] framework with an HTTP API for running tests across a cluster.

## Development

### Prerequisites

* [Virtualbox]
* [Docker Machine]
* [Docker Compose]

**Note:** In the [Docker Compose] installation guide, it mentions you will need
to install Docker first. This is *not* true in our scenario, so please skip the
instruction to install Docker, and just continue the installation by installing
Compose only.

### Before you get Started

Hold on there cowboy! You need to spin up a machine for these containers to run
on:

```bash
docker-machine create -d virtualbox --virtualbox-memory 4000 dev
```

### Workflow

Anytime you are in a shell and have not run the following commands, you
**must** do so before you can proceed:

```bash
docker-machine start dev
eval "$(docker-machine env dev)"
export COMPOSE_FILE=development.yml
```

Then from within the root directory of this project, you can write code and
start/restart all of the services for new changes to take effect:

```bash
docker-compose kill
docker-compose rm -f
docker-compose build && docker-compose up -d
docker-compose ps
```

You can then tail logs with `docker-compose logs`.

[Mesos]: http://mesos.apache.org/
[Virtualbox]: https://www.virtualbox.org
[Docker Machine]: https://docs.docker.com/machine/#installation
[Docker Compose]: https://docs.docker.com/compose/install/
