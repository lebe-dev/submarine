# Installation

**Language:** **EN** | [RU](README.ru.md) | [DE](README.de.md) | [ES](README.es.md)

## macOS

Homebrew ([lebe-dev tap](https://github.com/lebe-dev/homebrew-tap)):

```bash
brew install lebe-dev/tap/submarine
```

## Linux

```bash
curl -L -o sm-0.15.0-linux-amd64.zip \
  https://github.com/lebe-dev/submarine/releases/download/0.15.0/sm-0.15.0-linux-amd64.zip
unzip sm-0.15.0-linux-amd64.zip
sudo install -m 0755 sm /usr/local/bin/sm
sm --help
```

## Docker

The image is published as [`tinyops/submarine`](https://hub.docker.com/r/tinyops/submarine). Mount the directory with your subtitles into the container so `sm` can read and write the files:

```bash
docker run --rm -v "$PWD:/data" -w /data tinyops/submarine:0.15.0 --help

# Example: show info about a file in the current directory
docker run --rm -v "$PWD:/data" -w /data tinyops/submarine:0.15.0 \
  info Resident.Alien.S01E01.srt
```

To keep created files (e.g. backups) owned by your user instead of the container user, pass `-u`:

```bash
docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/data" -w /data \
  tinyops/submarine:0.15.0 set Resident.Alien.S01E01.srt 123 --text "Okay"
```

A convenient shell alias:

```bash
alias sm='docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/data" -w /data tinyops/submarine:0.15.0'
sm get Resident.Alien.S01E01.srt 123
```
