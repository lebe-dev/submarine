# Submarine

Submarine es un toolkit de asistencia para la traducción de subtítulos con LLM.

![Logo de Submarine](logo.png)

**Idioma:** [EN](README.md) | [RU](README.ru.md) | [DE](README.de.md) | **ES**

## Motivación

Prefiero ver películas, dibujos animados y anime en audio original. Sin embargo, los subtítulos en mi idioma suelen no estar disponibles. Aunque los LLM ya permiten traducirlos, no son perfectos: a veces mezclan numeraciones o marcas de tiempo. Incluso los mejores modelos cometen errores con frecuencia.

Además de la falta de subtítulos, estos pueden ser incompletos o tener problemas de compatibilidad con distintas versiones del lanzamiento.
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

## Casos de uso

Recetas orientadas a tareas para problemas reales con subtítulos — sincronización, fusión, comparación
y limpieza — con comandos listos para copiar y salida esperada. Índice completo —
[docs/usecases/](docs/usecases/README.es.md), incluyendo:

- [Fusionar una traducción incompleta con un archivo donante](docs/usecases/merge-incomplete-translation-with-donor.es.md)
- [Detectar y corregir un desplazamiento de sincronía constante](docs/usecases/detect-and-fix-constant-offset.es.md)
- [Corregir la deriva de velocidad de fotogramas (23.976 ↔ 25)](docs/usecases/fix-framerate-drift-with-rescale.es.md)
- [Resincronizar solo una parte del archivo](docs/usecases/resync-a-partial-range.es.md)
- [Unir partes divididas (CD1/CD2) en un solo archivo](docs/usecases/join-split-parts-cd1-cd2.es.md)
- [Comparar dos pistas de subtítulos](docs/usecases/compare-two-subtitle-tracks.es.md)
- [Normalizar la estructura: ordenar, renumerar, corregir solapamientos](docs/usecases/normalize-structure.es.md)
- [Eliminar duplicados y solapamientos tras una fusión](docs/usecases/deduplicate-and-clean-after-merge.es.md)
- [Encontrar diálogos faltantes con análisis de huecos](docs/usecases/find-missing-dialogue-gaps.es.md)

## Uso como biblioteca

Además de la interfaz de línea de comandos, Submarine puede usarse como biblioteca en proyectos Rust propios. Para más detalles sobre la integración, consulte la [documentación de la biblioteca](docs/LIBRARY.md).

## Hoja de ruta

- Feature: sincronización
- Feature: fusión
