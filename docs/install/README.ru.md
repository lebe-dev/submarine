# Установка

**Язык:** [EN](README.md) | **RU** | [DE](README.de.md) | [ES](README.es.md)

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

Образ публикуется как [`tinyops/submarine`](https://hub.docker.com/r/tinyops/submarine). Примонтируйте каталог с субтитрами внутрь контейнера, чтобы `sm` мог читать и записывать файлы:

```bash
docker run --rm -v "$PWD:/data" -w /data tinyops/submarine:0.15.0 --help

# Пример: показать информацию о файле из текущего каталога
docker run --rm -v "$PWD:/data" -w /data tinyops/submarine:0.15.0 \
  info Resident.Alien.S01E01.srt
```

Чтобы создаваемые файлы (например, резервные копии) принадлежали вашему пользователю, а не пользователю контейнера, передайте `-u`:

```bash
docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/data" -w /data \
  tinyops/submarine:0.15.0 set Resident.Alien.S01E01.srt 123 --text "Okay"
```

Удобный алиас для оболочки:

```bash
alias sm='docker run --rm -u "$(id -u):$(id -g)" -v "$PWD:/data" -w /data tinyops/submarine:0.15.0'
sm get Resident.Alien.S01E01.srt 123
```
