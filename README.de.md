# Submarine

Submarine ist ein kleines Toolkit zur LLM-gestützten Übersetzung von Untertiteln.

![Submarine Toolkit Logo](logo.png)

**Sprache:** [EN](README.md) | [RU](README.ru.md) | **DE** | [ES](README.es.md)

## Motivation

Ich schaue Filme, Cartoons und Anime am liebsten im Original. Untertitel in meiner Sprache sind jedoch oft nicht verfügbar. LLMs ermöglichen zwar die Übersetzung von Untertiteln, sind aber nicht fehlerfrei — manchmal verwechseln sie Nummerierungen oder Zeitstempel. Selbst die besten Modelle scheitern regelmäßig.

Submarine unterstützt den Übersetzungsprozess durch verschiedene Bearbeitungs- und Validierungswerkzeuge. So wird sichergestellt, dass übersetzte Untertitel korrekt und konsistent mit dem Original sind.

## Funktionen

- Untertitelformat: [SubRip](https://en.wikipedia.org/wiki/SubRip) (srt)
- Unterstützte Workflows:
  - Agent: [Eng](docs/AGENT-TRANSLATION-FLOW.md), [Rus](docs/AGENT-TRANSLATION-FLOW.RU.md), [De](docs/AGENT-TRANSLATION-FLOW.DE.md), [Es](docs/AGENT-TRANSLATION-FLOW.ES.md)
  - Chatbot: [Eng](docs/CHATBOT-TRANSLATION-FLOW.md), [Rus](docs/CHATBOT-TRANSLATION-FLOW.RU.md), [De](docs/CHATBOT-TRANSLATION-FLOW.DE.md), [Es](docs/CHATBOT-TRANSLATION-FLOW.ES.md)
- **Werkzeuge:**
  - Untertitel nach Index oder Bereich abrufen
  - Neuen Untertitel hinzufügen
  - Untertitel-Eigenschaften ändern
  - Untertitel aus Datei importieren (CSV, Anker-Format)
  - Zeitstempel mit Versatz anpassen
  - Untertiteldateien umbenennen (Stapelverarbeitung)
  - Untertitel im Anker-Format exportieren
  - Dateiprobleme diagnostizieren und beheben (`doctor`)
- **Verifikation:**
  - Übersetzte Untertitel gegen das Original prüfen
  - Übersetzungsfortschritt verfolgen
- **Agentenfreundlich:**
  - Strukturierte JSON-Ausgabe (`--output json`) für alle Befehle
  - Vorschau der Änderungen (`--dry-run`) für alle verändernden Befehle
  - Schema-Introspektion (`sm describe`)
  - Maschinenlesbare Fehlercodes und Hinweise
- **Automatische Sicherungen:** erstellt vor jeder Änderung automatisch Backups der Untertiteldateien.

## Installation

### macOS

Homebrew ([lebe-dev tap](https://github.com/lebe-dev/homebrew-tap)):

```bash
brew install lebe-dev/tap/submarine
```

### Linux

```bash
curl -L -o sm-0.14.0-linux-amd64.zip \
  https://github.com/lebe-dev/submarine/releases/download/0.14.0/sm-0.14.0-linux-amd64.zip
unzip sm-0.14.0-linux-amd64.zip
sudo install -m 0755 sm /usr/local/bin/sm
sm --help
```

## Verwendung

```bash
# Informationen zur Untertiteldatei anzeigen
# sm info [FILE.srt]
$ sm info Resident.Alien.S01E01.srt

# Untertitel nach Index oder Bereich abrufen
# sm get [FILE.srt] [INDEX or RANGE]
$ sm get Resident.Alien.S01E01.srt 123

123
00:06:54,111 --> 00:06:56,111
First subtitle

# Bereichssyntax wird ebenfalls unterstützt
$ sm get Resident.Alien.S01E01.srt 123-124

123
00:06:54,111 --> 00:06:56,111
First subtitle

124
00:06:56,111 --> 00:06:57,678
Second subtitle

# Untertitel für einen Index setzen
# sm set [--dry-run] [FILE.srt] [INDEX] \
#       [--start=00:00:03,481] \
#       [--end=00:00:04,481] \
#       [--text "TEXT"]
$ sm set Resident.Alien.S01E01.srt 123 \
       --text "Okay"

# Vorschau der Änderungen ohne Dateiänderung
$ sm set --dry-run Resident.Alien.S01E01.srt 123 --text "Okay"

# Untertitel am Ende der Datei hinzufügen
# Erhöht den Index automatisch und erstellt ein Backup
# Erstellt die srt-Datei, falls sie nicht existiert
# sm add [--dry-run] [FILE.srt] [START-END-TIMESTAMP] "[NEW-SUBTITLE]"
$ sm add Resident.Alien.S01E01.srt "00:03:03,481-00:03:04,481" "Okay"

# Zeitstempel um einen Versatz verschieben
# Unterstützt positive und negative Werte in Millisekunden
# sm delay [--dry-run] [FILE.srt] [OFFSET]
$ sm delay Resident.Alien.S01E01.srt "+1000"  # 1 Sekunde addieren
$ sm delay Resident.Alien.S01E01.srt "-500"   # 0,5 Sekunden subtrahieren

# Untertitel aus einer CSV-Datei importieren
# Erstellt die srt-Datei, falls sie nicht existiert
# sm import [--dry-run] [--format=csv,anchored] [--force] [FILE.srt] [IMPORT.csv]
$ sm import --format=csv Resident.Alien.S01E01.srt import.csv
$ sm import --format=anchored Resident.Alien.S01E01.srt import.txt

# Dateiintegrität prüfen
# sm doctor [--fix] [FILE.srt]
sm doctor --fix Resident.Alien.S01E01.eng.srt

# Stapelumbenennung
# - file-mask ist nicht zwischen Groß- und Kleinschreibung
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

# Untertitel im interaktiven Modus vergleichen
# sm compare [FILE1.srt] [FILE2.srt]
$ sm compare Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt

# Untertiteldateien prüfen
# sm verify [--range=1-50] [REFERENCE-FILE] [FILE2]
$ sm verify Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt
$ sm verify --range=1-50 Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt

Ergebnisse
==================

Matched: 874/876 (99.8%)
Missing in Resident.Alien.S01E01.rus.srt: 2
Index offset detected: -2
Missing subtitles:
  [848] 00:41:39,497 --> 00:41:42,325 (not found in Resident.Alien.S01E01.rus.srt)
  [...] ...

# Übersetzungsfortschritt anzeigen
# sm ts --reference [REFERENCE-FILE] [FILE2]
$ sm ts --reference Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt
Progress: 873/876 (99.7%)
Next chunk: 474-523

# Untertitel im angegebenen Format exportieren
# sm export [--format=anchored] [FILE.srt] [RANGE]
$ sm export --format=anchored movie.eng.srt 1-50

[1] Hello, how are you?
[2] I'm fine, thanks.
[3] Good to hear.
...
[50] See you tomorrow.

# JSON-Ausgabe (für alle Befehle außer compare)
$ sm get Resident.Alien.S01E01.srt 1 --output json
{"ok":true,"data":{"index":1,"start_time":"00:00:01,436",...}}

$ sm info Resident.Alien.S01E01.srt --output json

# Verfügbare Befehle und ihre Schemas anzeigen
$ sm describe
$ sm describe get
```

## Als Bibliothek verwenden

Neben der Befehlszeilenschnittstelle kann Submarine als Bibliothek in eigenen Rust-Projekten verwendet werden. Weitere Informationen zur Integration finden Sie in der [Bibliotheksdokumentation](docs/LIBRARY.md).

## Roadmap

- Feature: Synchronisierung
- Feature: Zusammenführen
