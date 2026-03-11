# Übersetzungs-Workflow mit Agent

Wie man Untertitel mit dem submarine-Tool und einem LLM-Agent (Codex, Claude Code, opencode usw.) von einer Sprache in eine andere übersetzt.

Zum Beispiel möchten Sie Episode 8 der Season 3 von Resident Alien ins Russische übersetzen.

**Sprache:** [EN](AGENT-TRANSLATION-FLOW.md) | [RU](AGENT-TRANSLATION-FLOW.RU.md) | **DE** | [ES](AGENT-TRANSLATION-FLOW.ES.md)

**Voraussetzungen:**
- Submarine CLI installiert
- Agent installiert
- Englische Untertiteldatei für Resident Alien Season 3

Fügen Sie in `CLAUDE.md` / `AGENTS.md` / `GEMINI.md` ein:

###[AGENTS.md]#######################################################

```markdown
## Submarine-Übersetzungs-Workflow

Sie übersetzen Untertitel von Englisch nach Russisch mit dem `sm` CLI-Tool.

Verwenden Sie `--output json` bei allen Befehlen für strukturierte Ausgabe.
Verwenden Sie `sm describe`, um verfügbare Befehle und ihre Parameter zu erkunden.

### Schritt 1: Status prüfen

touch Resident.Alien.S03E08.rus.srt

sm translation-status --output json --reference=Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt

Dies zeigt den Fortschritt und schlägt den nächsten Chunk-Bereich vor.

### Schritt 2: Chunk zur Übersetzung extrahieren

sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100

Ausgabeformat:

[1] Englischer Text hier
[2] Weiterer englischer Text
...

### Schritt 3: Übersetzen und in temporäre Datei speichern

Erstellen Sie die Datei `chunk.txt` mit Übersetzungen im selben Format:

[1] Russische Übersetzung
[2] Weiterer russischer Text
...

WICHTIG: Behalten Sie ALLE Indizes. Überspringen Sie keine Zeile.

### Schritt 4: Vorschau und Import der Übersetzungen

sm import --dry-run --reference=Resident.Alien.S03E08.eng.srt --format=anchored Resident.Alien.S03E08.rus.srt chunk.txt

Wenn die Vorschau korrekt aussieht:

sm import --force --reference=Resident.Alien.S03E08.eng.srt --format=anchored Resident.Alien.S03E08.rus.srt chunk.txt

### Schritt 5: Chunk überprüfen

sm verify --output json --range=1-100 Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt

Falls fehlende Indizes gemeldet werden, extrahieren Sie nur diese:

sm export --format=anchored Resident.Alien.S03E08.eng.srt 40-41

Übersetzen Sie die fehlenden Zeilen, hängen Sie sie an `chunk.txt` an und importieren Sie erneut.

### Schritt 6: Für den nächsten Chunk wiederholen

Löschen Sie `chunk.txt`.

sm translation-status --output json --reference=Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt

Fortfahren, bis der Fortschritt 100 % beträgt.

### Regeln

- Chunk-Größe: 100 Untertitel
- Immer `--dry-run` vor dem Import verwenden, um Änderungen zu prüfen
- Nach jedem Import immer verifizieren
- Alle fehlenden Einträge beheben, bevor der nächste Chunk beginnt
- Originalbedeutung bewahren, Redewendungen natürlich anpassen
- `--output json` für maschinenlesbare Ausgabe verwenden
```

###[/AGENTS.md]#######################################################

Starten Sie den Agent und geben Sie ihm folgenden Prompt:

```prompt
Lass uns die Untertitel für Resident Alien Season 3 Episode 8 übersetzen.
Nimm die englischen Untertitel aus `Resident.Alien.S03E08.eng.srt` und verwende das `sm`-Tool.

Ich erwarte `Resident.Alien.S03E08.rus.srt` als Ergebnis.
```
