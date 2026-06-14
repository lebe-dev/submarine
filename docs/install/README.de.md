# Installation

**Sprache:** [EN](README.md) | [RU](README.ru.md) | **DE** | [ES](README.es.md)

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

Das Image wird als [`tinyops/submarine`](https://hub.docker.com/r/tinyops/submarine) veröffentlicht. Binden Sie das Verzeichnis mit Ihren Untertiteln in den Container ein, damit `sm` die Dateien lesen und schreiben kann:

```bash
docker run --rm -v "$PWD:/data" -w /data tinyops/submarine:0.15.0 --help

# Beispiel: Informationen zu einer Datei im aktuellen Verzeichnis anzeigen
docker run --rm -v "$PWD:/data" -w /data tinyops/submarine:0.15.0 \
  info Resident.Alien.S01E01.srt
```

Damit erstellte Dateien (z. B. Backups) Ihrem Benutzer und nicht dem Container-Benutzer gehören, übergeben Sie `-u`:

```bash
docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/data" -w /data \
  tinyops/submarine:0.15.0 set Resident.Alien.S01E01.srt 123 --text "Okay"
```

Ein praktischer Shell-Alias:

```bash
alias sm='docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/data" -w /data tinyops/submarine:0.15.0'
sm get Resident.Alien.S01E01.srt 123
```
