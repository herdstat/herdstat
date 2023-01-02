# herdstat

[![stability-wip](https://img.shields.io/badge/stability-wip-lightgrey.svg)](https://github.com/mkenney/software-guides/blob/master/STABILITY-BADGES.md#work-in-progress)

> **Warning**
> `herdstat` is work in progress and neither feature complete nor tested thoroughly.

`herdstat` is a tool for analyzing and visualizing metrics of Open Source projects hosted on GitHub. As of today its
functionality is limited to generate GitHub-style contribution graphs for individual repository or complete GitHub
organisations.

## Namesake

`herdstat` is composed of the words _herd_, which means a group of people who usually have a common bond, and _stat_,
which is a well-known Linux command line utility that displays detailed information about files. So replacing _file_
with _open source community_ (= _herd_) does the trick in understanding why we have chosen that name.

## Build

The easiest way to build `herdstat` is to use Docker. To build the image invoke

```shell
docker build . -t herdstat-dev
```

Now you can execute `herdstat` on the _herdstat_ GitHub organization using

```shell
docker rm herdstat-dev || true
docker run --name herdstat-dev --security-opt="apparmor=unconfined" \
  --cap-add=SYS_PTRACE -p 40000:40000 -it herdstat-dev \
  /herdstat contribution-graph herdstat
```

To extract the generated contribution graph invoke

```shell
docker cp $(docker ps -aqf "name=herdstat-dev"):/contribution-graph.svg .
```

## Debug

To remote debug `herdstat` running in a docker container invoke

```shell
docker rm herdstat-dev || true
docker run --name herdstat-dev --security-opt="apparmor=unconfined" \
  --cap-add=SYS_PTRACE -p 40000:40000 -it herdstat-dev \
  "/dlv --listen=:40000 --headless=true --api-version=2 --accept-multiclient exec /herdstat -- --verbose contribution-graph herdstat"
```

You can then connect via your IDE or from the commandline on port 40000.
