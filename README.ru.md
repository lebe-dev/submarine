# Submarine

Submarine — небольшой набор инструментов для помощи в переводе субтитров с использованием LLM.

![Логотип Submarine](logo.png)

**Язык:** [EN](README.md) | **RU** | [DE](README.de.md) | [ES](README.es.md)

## Мотивация

Я предпочитаю смотреть фильмы, мультфильмы и аниме в оригинальной озвучке. Однако субтитры на моём языке нередко отсутствуют. LLM уже позволяют переводить субтитры, но они не идеальны: иногда ломают нумерацию или временны́е метки. Даже лучшие модели допускают ошибки.

Submarine помогает в процессе перевода, предоставляя инструменты редактирования и проверки. Он гарантирует, что переведённые субтитры точны и согласованы с оригиналом.

## Возможности

- Формат субтитров: [SubRip](https://en.wikipedia.org/wiki/SubRip) (srt)
- Поддерживаемые сценарии работы:
  - Агент: [Eng](docs/AGENT-TRANSLATION-FLOW.md), [Rus](docs/AGENT-TRANSLATION-FLOW.RU.md)
  - Чат-бот: [Eng](docs/CHATBOT-TRANSLATION-FLOW.md), [Rus](docs/CHATBOT-TRANSLATION-FLOW.RU.md)
- **Инструменты:**
  - Получить субтитр по индексу
  - Добавить новый субтитр
  - Импортировать субтитры из файла
  - Обновить субтитр со смещением
  - Массовое переименование файлов субтитров
  - Экспорт субтитров в формате с якорями
- **Проверка:**
  - Сравнение переведённых субтитров с оригиналом
  - Отслеживание прогресса перевода
- **Автобэкапы:** автоматически создают резервные копии файлов перед внесением изменений.

## Установка

### macOS

Homebrew ([tap lebe-dev](https://github.com/lebe-dev/homebrew-tap)):

```bash
brew install lebe-dev/tap/submarine
```

### Linux

```bash
curl -L -o sm-0.13.1-linux-amd64.zip \
  https://github.com/lebe-dev/submarine/releases/download/0.13.1/sm-0.13.1-linux-amd64.zip
unzip sm-0.13.1-linux-amd64.zip
sudo install -m 0755 sm /usr/local/bin/sm
sm --help
```

## Использование

```bash
# Информация о файле субтитров
# sm info [FILE.srt]
$ sm info Resident.Alien.S01E01.srt

# Получить субтитр по индексу или диапазону
# sm get [FILE.srt] [INDEX or RANGE]
$ sm get Resident.Alien.S01E01.srt 123

123
00:06:54,111 --> 00:06:56,111
First subtitle

# Поддерживается синтаксис диапазона
$ sm get Resident.Alien.S01E01.srt 123-124

123
00:06:54,111 --> 00:06:56,111
First subtitle

124
00:06:56,111 --> 00:06:57,678
Second subtitle

# Установить субтитр по индексу
# sm set [FILE.srt] [INDEX] \
#       [--start=00:00:03,481] \
#       [--end=00:00:04,481] \
#       [--text "TEXT"]
$ sm set Resident.Alien.S01E01.srt 123 \
       --text "Okay"

# Добавить субтитр в конец файла
# Автоматически увеличивает индекс и создаёт бэкап
# Создаёт srt-файл, если он не существует
# sm add [FILE.srt] [START-END-TIMESTAMP] "[NEW-SUBTITLE]"
$ sm add Resident.Alien.S01E01.srt "00:03:03,481-00:03:04,481" "Okay"

# Сдвинуть временны́е метки на заданное смещение
# Поддерживает положительные и отрицательные значения в миллисекундах
# sm delay [FILE.srt] [OFFSET]
$ sm delay Resident.Alien.S01E01.srt "+1000"  # Добавить 1 секунду
$ sm delay Resident.Alien.S01E01.srt "-500"   # Убрать 0.5 секунды

# Импортировать субтитры из csv-файла
# Создаёт srt-файл, если он не существует
# sm import [--dry-run] [--format=csv,anchored] [--force] [FILE.srt] [IMPORT.csv]
$ sm import --format=csv Resident.Alien.S01E01.srt import.csv
$ sm import --format=anchored Resident.Alien.S01E01.srt import.txt

# Проверить целостность файла
# sm doctor [--fix] [FILE.srt]
sm doctor --fix Resident.Alien.S01E01.eng.srt

# Массовое переименование
# - file-mask нечувствителен к регистру
# sm mass-rename [--dry-run] [--force] [--name="Resident Alien"] \
#          [--series-mode] [--season=3] \
#          [--language="rus"] \
#          [--separator="."] \
#          [--file-template="{{ name }}{{ separator }}S{{ season }}{{ separator }}E{{ episode }}.srt"] \
#          [FILE-MASK]
$ sm mass-rename --dry-run \
          --name="Resident Alien" \
          --series-mode --season=3 \
          --separator="." \
          "Resident"

# Сравнить субтитры в интерактивном режиме
# sm compare [FILE1.srt] [FILE2.srt]
$ sm compare Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt

# Проверить файлы субтитров
# sm verify [--range=1-50] [REFERENCE-FILE] [FILE2]
$ sm verify Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt
$ sm verify --range=1-50 Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt

Результаты
==================

Matched: 874/876 (99.8%)
Missing in Resident.Alien.S01E01.rus.srt: 2
Index offset detected: -2
Missing subtitles:
  [848] 00:41:39,497 --> 00:41:42,325 (not found in Resident.Alien.S01E01.rus.srt)
  [...] ...

# Прогресс перевода
# sm ts --reference [REFERENCE-FILE] [FILE2]
$ sm ts --reference Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt
Progress: 873/876 (99.7%)
Next chunk: 474-523

# Экспортировать субтитры в указанном формате
# sm export [--format=anchored] [FILE.srt] [RANGE]
$ sm export --format=anchored movie.eng.srt 1-50

[1] Hello, how are you?
[2] I'm fine, thanks.
[3] Good to hear.
...
[50] See you tomorrow.
```

## Использование как библиотека

Помимо интерфейса командной строки, Submarine можно использовать как библиотеку в собственных Rust-проектах. Подробности — в [документации библиотеки](docs/LIBRARY.md).

## Дорожная карта

- Рефакторинг кода
- Функция: синхронизация
- Функция: слияние
