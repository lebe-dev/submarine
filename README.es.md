# Submarine

Submarine es un toolkit de asistencia para la traducción de subtítulos con LLM.

![Logo de Submarine](logo.png)

**Idioma:** [EN](README.md) | [RU](README.ru.md) | [DE](README.de.md) | **ES**

## Motivación

Prefiero ver películas, dibujos animados y anime en audio original. Sin embargo, los subtítulos en mi idioma suelen no estar disponibles. Aunque los LLM ya permiten traducirlos, no son perfectos: a veces mezclan numeraciones o marcas de tiempo. Incluso los mejores modelos cometen errores con frecuencia.

Submarine está diseñado para ayudar en el proceso de traducción con herramientas de edición y validación, garantizando que los subtítulos traducidos sean precisos y coherentes con el contenido original.

## Características

- Formato compatible: [SubRip](https://en.wikipedia.org/wiki/SubRip) (srt)
- Flujos de trabajo compatibles:
  - [Agente](docs/AGENT-TRANSLATION-FLOW.ES.md) (Recomendado)
  - [Chatbot](docs/CHATBOT-TRANSLATION-FLOW.ES.md)
- **Herramientas:**
  - Obtener subtítulo por índice o rango
  - Añadir un nuevo subtítulo
  - Modificar propiedades de un subtítulo
  - Importar subtítulos desde archivo (CSV, formato anclado)
  - Ajustar marcas de tiempo con desplazamiento
  - Renombrado masivo de archivos de subtítulos
  - Exportar subtítulos en formato anclado
  - Diagnosticar y reparar problemas del archivo (`doctor`)
- **Verificación:**
  - Verificar subtítulos traducidos contra el original
  - Seguimiento del progreso de traducción
- **Adaptado para agentes:**
  - Salida JSON estructurada (`--output json`) en todos los comandos
  - Vista previa de cambios (`--dry-run`) en todos los comandos que modifican archivos
  - Introspección del esquema (`sm describe`)
  - Códigos de error y sugerencias legibles por máquinas
- **Copias de seguridad automáticas:** crea backups de los archivos de subtítulos antes de realizar cambios.

## Instalación

macOS (Homebrew), Linux, Docker — ver [docs/install/](docs/install/README.es.md).

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
# sm set [--dry-run] [FILE.srt] [INDEX] \
#       [--start=00:00:03,481] \
#       [--end=00:00:04,481] \
#       [--text "TEXT"]
$ sm set Resident.Alien.S01E01.srt 123 \
       --text "Okay"

# Vista previa sin modificar el archivo
$ sm set --dry-run Resident.Alien.S01E01.srt 123 --text "Okay"

# Añadir subtítulo al final del archivo
# Incrementa el índice automáticamente y crea un backup
# Crea el archivo srt si no existe
# sm add [--dry-run] [FILE.srt] [START-END-TIMESTAMP] "[NEW-SUBTITLE]"
$ sm add Resident.Alien.S01E01.srt "00:03:03,481-00:03:04,481" "Okay"

# Ajustar marcas de tiempo con un desplazamiento
# Admite valores positivos y negativos en milisegundos
# sm delay [--dry-run] [FILE.srt] [OFFSET]
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

# Salida JSON (disponible en todos los comandos excepto compare)
$ sm get Resident.Alien.S01E01.srt 1 --output json
{"ok":true,"data":{"index":1,"start_time":"00:00:01,436",...}}

$ sm info Resident.Alien.S01E01.srt --output json

# Descubrir los comandos disponibles y sus esquemas
$ sm describe
$ sm describe get
```

## Casos de uso

Recetas orientadas a tareas para problemas reales con subtítulos — sincronización, fusión, comparación
y limpieza — con comandos listos para copiar y salida esperada. Consulte el índice completo en
[docs/usecases/](docs/usecases/README.es.md), incluyendo:

- [Fusionar una traducción incompleta con un archivo donante](docs/usecases/merge-incomplete-translation-with-donor.es.md)
- [Detectar y corregir un desplazamiento de sincronía constante](docs/usecases/detect-and-fix-constant-offset.es.md)
- [Corregir la deriva de velocidad de fotogramas (23.976 ↔ 25)](docs/usecases/fix-framerate-drift-with-rescale.es.md)

## Uso como biblioteca

Además de la interfaz de línea de comandos, Submarine puede usarse como biblioteca en proyectos Rust propios. Para más detalles sobre la integración, consulte la [documentación de la biblioteca](docs/LIBRARY.md).

## Hoja de ruta

- Feature: sincronización
- Feature: fusión
