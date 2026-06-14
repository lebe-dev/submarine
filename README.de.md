# Submarine

Submarine ist ein Toolkit zur LLM-gestützten Übersetzung von Untertiteln.

![Submarine Toolkit Logo](logo.png)

**Sprache:** [EN](README.md) | [RU](README.ru.md) | **DE** | [ES](README.es.md)

## Motivation

Ich schaue Filme, Cartoons und Anime am liebsten im Original. Untertitel in meiner Sprache sind jedoch oft nicht verfügbar. LLMs ermöglichen zwar die Übersetzung von Untertiteln, sind aber nicht fehlerfrei — manchmal verwechseln sie Nummerierungen oder Zeitstempel. Selbst die besten Modelle scheitern regelmäßig.

Neben fehlenden Untertiteln können diese auch unvollständig sein oder Kompatibilitätsprobleme mit verschiedenen Releases haben.
Submarine unterstützt den Übersetzungsprozess durch verschiedene Bearbeitungs- und Validierungswerkzeuge. So wird sichergestellt, dass übersetzte Untertitel korrekt und konsistent mit dem Original sind.

## Funktionen

- Unterstütztes Format: [SubRip](https://en.wikipedia.org/wiki/SubRip) (srt)
- Unterstützte Workflows:
  - [Agent](docs/AGENT-TRANSLATION-FLOW.DE.md) (Empfohlen)
  - [Chatbot](docs/CHATBOT-TRANSLATION-FLOW.DE.md)
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

macOS (Homebrew), Linux, Docker — siehe [docs/install/](docs/install/README.de.md).

## Anwendungsfälle

Aufgabenorientierte Rezepte für häufige Untertitelprobleme — Synchronisierung, Zusammenführen,
Vergleichen und Bereinigen — mit kopierfertigen Befehlen und erwartetem Ausgabe. Vollständiger Index —
[docs/usecases/](docs/usecases/README.de.md), darunter:

- [Unvollständige Übersetzung mit einer Spenderdatei zusammenführen](docs/usecases/merge-incomplete-translation-with-donor.de.md)
- [Konstanten Sync-Versatz erkennen und beheben](docs/usecases/detect-and-fix-constant-offset.de.md)
- [Bildfrequenz-Drift korrigieren (23.976 ↔ 25)](docs/usecases/fix-framerate-drift-with-rescale.de.md)

## Als Bibliothek verwenden

Neben der Befehlszeilenschnittstelle kann Submarine als Bibliothek in eigenen Rust-Projekten verwendet werden. Weitere Informationen zur Integration finden Sie in der [Bibliotheksdokumentation](docs/LIBRARY.md).

## Roadmap

- Feature: Synchronisierung
- Feature: Zusammenführen
