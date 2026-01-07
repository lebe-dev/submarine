# Процесс перевода субтитров с помощью чат-бота

Как переводить субтитры с одного языка на другой с помощью инструмента submarine и LLM-чат-бота (ChatGPT, Claude, DeepSeek и т.д.).

Например, вы хотите перевести 8-ю серию 3-го сезона сериала "Засланец из космоса" (Resident Alien) на русский язык.

**Требования:**
- Установленный Submarine CLI
- Файл английских субтитров для Resident Alien Season 3

Сначала необходимо экспортировать субтитры в специальном формате (anchored). Этот формат подходит для любых популярных LLM, так как исходный формат SubRip слишком сложен для современных языковых моделей (ломаются timestamps, пропускаются индексы).

Экспортируйте фрагмент субтитров и скопируйте в буфер обмена:

```bash
# Linux (X11)
sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100 | xclip

# Linux (Wayland)
sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100 | wl-copy

# MacOS
sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100 | pbcopy
```

Подготовьте начальный промпт для чат-бота следующим образом:

```prompt
Давайте переведем субтитры с английского на русский для сериала "Резидент-инопланетянин", 3 сезон, 8 серия.

Я предоставлю субтитры в специальном формате, ваша задача — сохранить формат и перевести их на русский.

Правила:
- Сохраняйте формат [ИНДЕКС] текст
- Переводите ВСЕ строки, не пропускайте ни одной
- Не добавляйте объяснений, только перевод

Пример формата:
[1] Привет, как дела?
[2] Хорошо, спасибо.
[3] Что привело тебя сюда сегодня?
...
[50] Увидимся завтра.

### СОДЕРЖАНИЕ ###

[1] <i>Ранее в "Резиденте-инопланетянине"...</i>
[2] Залезай!
[3] Может, он не мертв.
- Что?
[4] Мы должны вернуться и проверить.
[5] <i>Джозеф — инопланетянин.</i>
[6] Они существуют.
[7] <i>Они похищают людей,</i>
[8] <i>и те ничего не помнят.</i>
[9] Нет!
[10] Хотя бы в этот раз
дай мне запомнить.
...
```

Вставьте промпт в чатбот и дождитесь перевода.

Затем создайте файл `chatbot-import.md` и добавьте в него переведенное содержимое:

```bash
touch chatbot-import.md

# Linux (X11)
xclip -selection clipboard -o >> chatbot-import.md

# Linux (Wayland)
wl-paste >> chatbot-import.md

# MacOS
pbpaste >> chatbot-import.md
```

Либо отредактируйте файл вручную.

Затем скормите чатботу следующие фрагменты `101-200`, `201-300` и так до конца. Всегда добавляйте содержимое в файл `chatbot-import.md`.

Наконец, создайте итоговый файл субтитров:

```bash
touch Resident.Alien.S03E08.rus.srt

sm import --format=anchored --reference=Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt chatbot-import.md
```

И проверьте результат:

```bash
sm verify --format=anchored Resident.Alien.S03E08.rus.srt Resident.Alien.S03E08.eng.srt

Verifying subtitle files
========================

Reference file: Resident.Alien.S03E08.eng.srt (876 subtitles)
Target file:    Resident.Alien.S03E08.rus.srt (876 subtitles)

Results
=======

Matched: 876/876 (100.0%)

Verification: SUCCESS
```
