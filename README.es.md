# Submarine

Submarine es un pequeño toolkit de asistencia para la traducción de subtítulos con LLM.

![Logo de Submarine](logo.png)

**Idioma:** [EN](README.md) | [RU](README.ru.md) | [DE](README.de.md) | **ES**

## Motivación

Prefiero ver películas, dibujos animados y anime en audio original. Sin embargo, los subtítulos en mi idioma suelen no estar disponibles. Aunque los LLM ya permiten traducirlos, no son perfectos: a veces mezclan numeraciones o marcas de tiempo. Incluso los mejores modelos cometen errores con frecuencia.

Submarine está diseñado para ayudar en el proceso de traducción con herramientas de edición y validación, garantizando que los subtítulos traducidos sean precisos y coherentes con el contenido original.

## Características

- Formato de subtítulos: [SubRip](https://en.wikipedia.org/wiki/SubRip) (srt)
- Flujos de trabajo compatibles:
  - Agente: [Eng](docs/AGENT-TRANSLATION-FLOW.md), [Rus](docs/AGENT-TRANSLATION-FLOW.RU.md)
  - Chatbot: [Eng](docs/CHATBOT-TRANSLATION-FLOW.md), [Rus](docs/CHATBOT-TRANSLATION-FLOW.RU.md)
- **Herramientas:**
  - Obtener subtítulo por índice
  - Añadir un nuevo subtítulo
  - Importar subtítulos desde archivo
  - Actualizar subtítulo con desplazamiento
  - Renombrado masivo de archivos de subtítulos
  - Exportar subtítulos en formato anclado
- **Verificación:**
  - Verificar subtítulos traducidos contra el original
  - Seguimiento del progreso de traducción
- **Copias de seguridad automáticas:** crea backups de los archivos de subtítulos antes de realizar cambios.

## Instalación

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

## Uso

```bash
# Mostrar información sobre el archivo de subtítulos
# sm info [FILE.srt]
$ sm info Resident.Alien.S01E01.srt

# Obtener subtítulo por índice o rango
# sm get [FILE.srt] [INDEX or RANGE]
$ sm get Resident.Alien.S01E01.srt 123

123
00:06:54,111 --> 00:06:56,111
First subtitle

# También admite sintaxis de rango
$ sm get Resident.Alien.S01E01.srt 123-124

123
00:06:54,111 --> 00:06:56,111
First subtitle

124
00:06:56,111 --> 00:06:57,678
Second subtitle

# Establecer subtítulo para un índice
# sm set [FILE.srt] [INDEX] \
#       [--start=00:00:03,481] \
#       [--end=00:00:04,481] \
#       [--text "TEXT"]
$ sm set Resident.Alien.S01E01.srt 123 \
       --text "Okay"

# Añadir subtítulo al final del archivo
# Incrementa el índice automáticamente y crea un backup
# Crea el archivo srt si no existe
# sm add [FILE.srt] [START-END-TIMESTAMP] "[NEW-SUBTITLE]"
$ sm add Resident.Alien.S01E01.srt "00:03:03,481-00:03:04,481" "Okay"

# Ajustar marcas de tiempo con un desplazamiento
# Admite valores positivos y negativos en milisegundos
# sm delay [FILE.srt] [OFFSET]
$ sm delay Resident.Alien.S01E01.srt "+1000"  # Añadir 1 segundo
$ sm delay Resident.Alien.S01E01.srt "-500"   # Restar 0.5 segundos

# Importar subtítulos desde un archivo csv
# Crea el archivo srt si no existe
# sm import [--dry-run] [--format=csv,anchored] [--force] [FILE.srt] [IMPORT.csv]
$ sm import --format=csv Resident.Alien.S01E01.srt import.csv
$ sm import --format=anchored Resident.Alien.S01E01.srt import.txt

# Verificar integridad del archivo
# sm doctor [--fix] [FILE.srt]
sm doctor --fix Resident.Alien.S01E01.eng.srt

# Renombrado masivo
# - file-mask no distingue mayúsculas de minúsculas
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

# Comparar subtítulos en modo interactivo
# sm compare [FILE1.srt] [FILE2.srt]
$ sm compare Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt

# Verificar archivos de subtítulos
# sm verify [--range=1-50] [REFERENCE-FILE] [FILE2]
$ sm verify Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt
$ sm verify --range=1-50 Resident.Alien.S01E01.eng.srt Resident.Alien.S01E01.rus.srt

Resultados
==================

Matched: 874/876 (99.8%)
Missing in Resident.Alien.S01E01.rus.srt: 2
Index offset detected: -2
Missing subtitles:
  [848] 00:41:39,497 --> 00:41:42,325 (not found in Resident.Alien.S01E01.rus.srt)
  [...] ...

# Progreso de traducción
# sm ts --reference [REFERENCE-FILE] [FILE2]
$ sm ts --reference Resident.Alien.S03E08.eng.srt Resident.Alien.S03E08.rus.srt
Progress: 873/876 (99.7%)
Next chunk: 474-523

# Exportar subtítulos en el formato especificado
# sm export [--format=anchored] [FILE.srt] [RANGE]
$ sm export --format=anchored movie.eng.srt 1-50

[1] Hello, how are you?
[2] I'm fine, thanks.
[3] Good to hear.
...
[50] See you tomorrow.
```

## Uso como biblioteca

Además de la interfaz de línea de comandos, Submarine puede usarse como biblioteca en proyectos Rust propios. Para más detalles sobre la integración, consulte la [documentación de la biblioteca](docs/LIBRARY.md).

## Hoja de ruta

- Refactorización del código
- Feature: sincronización
- Feature: fusión
