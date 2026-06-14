# Casos de uso

Recetas orientadas a tareas para problemas reales con subtítulos. Cada página describe un
problema y explica paso a paso los comandos `sm` exactos que lo resuelven — con archivos de
ejemplo listos para copiar y pegar, la salida esperada, una vista previa con `--dry-run`
antes de cualquier modificación y un ejemplo con `--output json` para scripting.

Estas son recetas, no una referencia de flags. Para la lista completa de opciones de
cualquier comando, usa `sm <comando> --help`; para esquemas en formato máquina, usa
`sm describe`.

## Por tarea

### Sincronización y tiempos

| Caso de uso | Soluciona | Comando principal |
|---|---|---|
| [Detectar y corregir un offset de sincronía constante](detect-and-fix-constant-offset.es.md) | Los subtítulos van consistentemente adelantados o retrasados en una cantidad fija | `detect-offset` → `delay` |
| [Corregir deriva de velocidad de fotogramas (23.976 ↔ 25)](fix-framerate-drift-with-rescale.es.md) | La sincronía se desvía progresivamente hacia el final (desajuste de fps) | `rescale` |
| [Resincronizar solo una parte del archivo](resync-a-partial-range.es.md) | El desplazamiento aparece solo después de cierto punto o empalme | `delay --range` / `--from-timestamp` |

### Fusión y unión

| Caso de uso | Soluciona | Comando principal |
|---|---|---|
| [Fusionar una traducción incompleta con un archivo donante](merge-incomplete-translation-with-donor.es.md) | Completar una traducción parcial desde otro archivo sin tocar las líneas de confianza | `merge` |
| [Unir partes divididas (CD1/CD2) en un solo archivo](join-split-parts-cd1-cd2.es.md) | Una publicación dividida en partes consecutivas que necesita unificarse | `concat` |
| [Eliminar duplicados y solapamientos tras una fusión](deduplicate-and-clean-after-merge.es.md) | Líneas casi duplicadas se acumulan tras una fusión agresiva | `dedupe` |

### Inspección y estructura

| Caso de uso | Soluciona | Comando principal |
|---|---|---|
| [Encontrar diálogos faltantes con análisis de huecos](find-missing-dialogue-gaps.es.md) | Localizar silencios largos donde puede faltar diálogo | `gaps` |
| [Comparar dos pistas de subtítulos](compare-two-subtitle-tracks.es.md) | Entender la diferencia de contenido entre dos pistas | `diff` |
| [Normalizar la estructura: ordenar, renumerar, corregir solapamientos](normalize-structure.es.md) | Un archivo válido quedó desordenado tras ediciones manuales | `normalize` |

## Convenciones usadas en estas recetas

- **`--out` es el archivo de resultado; `--output text|json` es el flag de formato global.**
  Son distintos — `merge`, `rescale` y `concat` escriben en `--out`.
- **Los comandos que modifican archivos guardan una copia de seguridad** en `backups/`
  antes de cambiar el archivo en su lugar, y todos admiten `--dry-run` para previsualizar.
- **Offsets negativos:** coloca los flags de opciones antes de un offset negativo
  (`sm delay --dry-run file -500`), ya que un `-` al inicio puede confundirse con un flag.

Consulta el [README](../../README.es.md) del proyecto para instrucciones de instalación y la lista completa de comandos.
