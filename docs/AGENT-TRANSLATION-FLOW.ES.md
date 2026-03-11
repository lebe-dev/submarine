# Flujo de traducción con agente

Cómo traducir subtítulos de un idioma a otro usando la herramienta submarine y un agente LLM (Codex, Claude Code, opencode, etc.).

Por ejemplo, quiere traducir el Episodio 8 de la Temporada 3 de Resident Alien al ruso.

**Idioma:** [EN](AGENT-TRANSLATION-FLOW.md) | [RU](AGENT-TRANSLATION-FLOW.RU.md) | [DE](AGENT-TRANSLATION-FLOW.DE.md) | **ES**

**Requisitos:**
- Submarine CLI instalado
- Agente instalado
- Archivo de subtítulos en inglés para Resident Alien Season 3

Coloque en `CLAUDE.md` / `AGENTS.md` / `GEMINI.md`:

###[AGENTS.md]#######################################################

```markdown
## Flujo de trabajo de traducción con Submarine

Está traduciendo subtítulos del inglés al ruso usando la herramienta CLI `sm`.

Use `--output json` en todos los comandos para obtener salida estructurada.
Use `sm describe` para descubrir los comandos disponibles y sus parámetros.

### Paso 1: Comprobar estado

touch Resident.Alien.S03E08.rus.srt

sm translation-status --output json --reference=Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt

Esto muestra el progreso y sugiere el siguiente rango de fragmento.

### Paso 2: Extraer fragmento para traducción

sm export --format=anchored Resident.Alien.S03E08.eng.srt 1-100

Formato de salida:

[1] Texto en inglés aquí
[2] Más texto en inglés
...

### Paso 3: Traducir y guardar en archivo temporal

Cree el archivo `chunk.txt` con las traducciones en el mismo formato:

[1] Traducción al ruso
[2] Más texto en ruso
...

IMPORTANTE: Mantenga TODOS los índices. No omita ninguna línea.

### Paso 4: Vista previa e importar traducciones

sm import --dry-run --reference=Resident.Alien.S03E08.eng.srt --format=anchored Resident.Alien.S03E08.rus.srt chunk.txt

Si la vista previa es correcta:

sm import --force --reference=Resident.Alien.S03E08.eng.srt --format=anchored Resident.Alien.S03E08.rus.srt chunk.txt

### Paso 5: Verificar fragmento

sm verify --output json --range=1-100 Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt

Si se informan índices faltantes, extráigalos solo esos:

sm export --format=anchored Resident.Alien.S03E08.eng.srt 40-41

Traduzca los faltantes, añádalos a `chunk.txt` e importe de nuevo.

### Paso 6: Repetir con el siguiente fragmento

Elimine `chunk.txt`.

sm translation-status --output json --reference=Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt

Continúe hasta que el progreso sea del 100 %.

### Reglas

- Tamaño del fragmento: 100 subtítulos
- Usar siempre `--dry-run` antes de importar para vista previa
- Verificar siempre después de importar
- Corregir todos los faltantes antes de pasar al siguiente fragmento
- Preservar el significado original, adaptar los modismos de forma natural
- Usar `--output json` para salida legible por máquinas
```

###[/AGENTS.md]#######################################################

Inicie el agente y proporciónele el siguiente prompt:

```prompt
Vamos a traducir los subtítulos de Resident Alien Season 3 Episode 8.
Toma los subtítulos en inglés de `Resident.Alien.S03E08.eng.srt` y usa la herramienta `sm`.

Espero `Resident.Alien.S03E08.rus.srt` como resultado.
```
