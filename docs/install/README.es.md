# Instalación

**Idioma:** [EN](README.md) | [RU](README.ru.md) | [DE](README.de.md) | **ES**

## macOS

Homebrew ([tap lebe-dev](https://github.com/lebe-dev/homebrew-tap)):

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

La imagen se publica como [`tinyops/submarine`](https://hub.docker.com/r/tinyops/submarine). Monte el directorio con sus subtítulos dentro del contenedor para que `sm` pueda leer y escribir los archivos:

```bash
docker run --rm -v "$PWD:/data" -w /data tinyops/submarine:0.15.0 --help

# Ejemplo: mostrar información de un archivo del directorio actual
docker run --rm -v "$PWD:/data" -w /data tinyops/submarine:0.15.0 \
  info Resident.Alien.S01E01.srt
```

Para que los archivos creados (por ejemplo, las copias de seguridad) pertenezcan a su usuario y no al usuario del contenedor, pase `-u`:

```bash
docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/data" -w /data \
  tinyops/submarine:0.15.0 set Resident.Alien.S01E01.srt 123 --text "Okay"
```

Un alias de shell práctico:

```bash
alias sm='docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/data" -w /data tinyops/submarine:0.15.0'
sm get Resident.Alien.S01E01.srt 123
```
