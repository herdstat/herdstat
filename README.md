# herdstat

[![stability-wip](https://img.shields.io/badge/stability-wip-lightgrey.svg)](https://github.com/mkenney/software-guides/blob/master/STABILITY-BADGES.md#work-in-progress)
[![codecov](https://codecov.io/gh/herdstat/herdstat/branch/main/graph/badge.svg?token=GG15UAXAYR)](https://codecov.io/gh/herdstat/herdstat)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/cd018680eedc4f6b88976356cd2647e8)](https://www.codacy.com/gh/herdstat/herdstat/dashboard?utm_source=github.com&utm_medium=referral&utm_content=herdstat/herdstat&utm_campaign=Badge_Grade)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-%23FE5196?logo=conventionalcommits&logoColor=white)](https://conventionalcommits.org)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit&logoColor=white)](https://github.com/pre-commit/pre-commit)

> **Warning** `herdstat` is work in progress and neither feature complete nor tested thoroughly.

`herdstat` is a tool for analyzing and visualizing metrics of Open Source projects hosted on GitHub. As of today its
functionality is limited to generate GitHub-style contribution graphs for individual repositories or whole GitHub
organisations.

## Namesake

`herdstat` is composed of the words _herd_, which means a group of people who usually have a common bond, and _stat_,
which is a well-known Linux command line utility that displays detailed information about files. So replacing _file_
with _open source community_ (= _herd_) does the trick in understanding why we have chosen that name.

## Usage

You can execute `herdstat` using Docker via

```shell
docker run herdstat/herdstat /herdstat -r herdstat contribution-graph
```

Alternatively, you can use the [`herdstat` GitHub action](https://github.com/herdstat/herdstat-action).

## Configuration

`herdstat` can be configured either by providing arguments to the CLI or by means of a configuration file via the global
`--config` CLI flag. The list of available configuration options is summarized in the following table:

| Aspect              | Subcommand         | Description                                                                                                           | CLI Flag                  | Configuration Path            |
| ------------------- | ------------------ | --------------------------------------------------------------------------------------------------------------------- | ------------------------- | ----------------------------- |
| Configuration       | -                  | Path to a configuration file (see [reference](.herdstat.reference.yaml)).                                             | `--config`, `-c`          | -                             |
| Source Repositories | -                  | The comma-delimited list of GitHub repositories to analyze. May be either single repositories or whole organizations. | `--repositories`, `-r`    | `repositories`                |
| Github Token        | -                  | Token used to access the GitHub API.                                                                                  | `--repositories`, `-r`    | `repositories`                |
| Verbosity           | -                  | Controls the verbosity of the `herdstat` CLI.                                                                         | `--github-token`, `-t`    | `github-token`                |
| Analysis Period     | contribution-graph | Controls the period of time to analyze by means of the last day of the 52 week period to look at.                     | `--until`, `-u`           | `contribution-graph/until`    |
| Minification        | contribution-graph | Whether to minify the generated SVG.                                                                                  | `--minify`, `-m`          | `contribution-graph/minify`   |
| Output Filename     | contribution-graph | The name of the file used to store the generated contribution graph.                                                  | `--output-filename`, `-o` | `contribution-graph/filename` |
| Primary Color       | contribution-graph | The primary color used for coloring daily contribution cells (hex-encoded RGB without leading '#').                   | `--color`                 | `contribution-graph/color`    |

## Building from Source

The easiest way to build `herdstat` from its sources is to use Docker. To build the image invoke

```shell
docker build . -t herdstat-dev
```

You can execute `herdstat` on the _herdstat_ GitHub organization using

```shell
docker rm herdstat-dev || true
docker run --name herdstat-dev -it herdstat-dev /herdstat -r herdstat contribution-graph
```

To extract the generated contribution graph from the Docker container invoke

```shell
docker cp $(docker ps -aqf "name=herdstat-dev"):/contribution-graph.svg .
```

## Debug

To remote debug `herdstat` build the image with the `ENV` build variable set to `debug`

```shell
docker build  -t herdstat-dev --build-arg ENV=debug .
```

and start a container using

```shell
docker rm herdstat-dev || true
docker run --name herdstat-dev --security-opt="apparmor=unconfined" \
  --cap-add=SYS_PTRACE -p 40000:40000 -it herdstat-dev \
  /dlv --listen=:40000 --headless=true --api-version=2 --accept-multiclient exec /herdstat -- --verbose -r herdstat contribution-graph
```

You can then connect via your IDE or from the commandline on port 40000.
