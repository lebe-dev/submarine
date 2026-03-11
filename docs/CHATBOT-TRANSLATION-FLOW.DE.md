# Übersetzungs-Workflow mit Chatbot

Wie man Untertitel mit dem submarine-Tool und einem LLM-Chatbot (ChatGPT, Claude, DeepSeek usw.) von einer Sprache in eine andere übersetzt.

Zum Beispiel möchten Sie Episode 8 der Season 3 von Resident Alien ins Russische übersetzen.

**Sprache:** [EN](CHATBOT-TRANSLATION-FLOW.md) | [RU](CHATBOT-TRANSLATION-FLOW.RU.md) | **DE** | [ES](CHATBOT-TRANSLATION-FLOW.ES.md)

**Voraussetzungen:**
- Submarine CLI installiert
- Englische Untertiteldatei für Resident Alien Season 3

Zunächst müssen wir die Untertitel in einem speziellen Format (anchored) exportieren. Dieses Format ist für alle gängigen LLMs geeignet, da das ursprüngliche SubRip-Format für moderne LLMs zu komplex ist.

Untertitel-Chunk exportieren und in die Zwischenablage kopieren:

```bash
# Linux (X11)
sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100 | xclip

# Linux (Wayland)
sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100 | wl-copy

# MacOS
sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100 | pbcopy
```

Erstellen Sie den initialen Prompt für den Chatbot:

```prompt
Lass uns Untertitel von Englisch nach Russisch für Resident Alien Season 3 Episode 8 übersetzen.

Ich gebe dir die Untertitel in einem speziellen Format. Deine Aufgabe ist es, das Format beizubehalten und sie ins Russische zu übersetzen.

Regeln:
- Format [INDEX] Text beibehalten
- ALLE Zeilen übersetzen, keine überspringen
- Keine Erklärungen hinzufügen, nur die Übersetzung

Formatbeispiel:
[1] Hello, how are you?
[2] I'm fine, thanks.
[3] What brings you here today?
...
[50] See you tomorrow.

### INHALT ###

[1] <i>Previously on "Resident Alien"...</i>
[2] Get in!
[3] Maybe he's not dead.
- What?
[4] We have to go back and see.
[5] <i>Joseph is an alien.</i>
[6] They're real.
[7] <i>They abduct people,</i>
[8] <i>and they don't remember.</i>
[9] No!
[10] At least let me remember
this time.
```

Erstellen Sie dann die Datei `chatbot-import.md` und hängen Sie den übersetzten Inhalt an:

```bash
touch chatbot-import.md

# Linux (X11)
xclip -selection clipboard -o >> chatbot-import.md

# Linux (Wayland)
wl-paste >> chatbot-import.md

# MacOS
pbpaste >> chatbot-import.md
```

Kopieren Sie dann die nächsten Chunks `101-200`, `201-300` bis zum Ende. Hängen Sie immer an die Datei `chatbot-import.md` an.

Erstellen Sie abschließend die Untertiteldatei:

```bash
touch Resident.Alien.S03E08.rus.srt

sm import --format=anchored --reference=Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt chatbot-import.md
```

Und überprüfen Sie das Ergebnis:

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
