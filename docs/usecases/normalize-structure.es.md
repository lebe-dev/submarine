# Normalizar la estructura: ordenar, renumerar, corregir solapamientos

> Tras ediciones manuales o por script, devuelve un SRT válido a su forma correcta — ordenado por tiempo, renumerado desde 1, con solapamientos recortados.

## Escenario

Las ediciones manuales o un script rápido dejaron el archivo desordenado: los índices ya
no son ascendentes, los fragmentos no están ordenados por tiempo de inicio y un par de
vecinos se solapan. El archivo todavía se puede parsear, pero está desordenado.
`sm normalize` lo reescribe en su forma canónica.

## Comandos utilizados

- `sm normalize` ★ — ordenar, renumerar y opcionalmente corregir solapamientos
- `sm doctor` — para el trabajo diferente de reparar errores estructurales de parseo

## Archivos de ejemplo

`scram.srt` — desordenado, con el primer fragmento solapándose con el segundo:

```srt
3
00:00:08,000 --> 00:00:10,000
Third in time.

1
00:00:01,000 --> 00:00:04,000
First, overlaps next.

2
00:00:03,000 --> 00:00:05,000
Second.
```

El fragmento `1` termina en `00:00:04,000` pero el fragmento `2` comienza en
`00:00:03,000` — un solapamiento de 1 segundo.

## Paso a paso

### 1. Previsualizar con `--dry-run`

```bash
sm normalize scram.srt --sort --renumber --fix-overlaps --dry-run
```

```
Dry-run: normalization would be applied

Sorted: yes (by start time)
Renumbered: yes (from 1)
Overlaps fixed: 1
Subtitles: 3
```

`Overlaps fixed: 1` es el par de vecinos que será recortado. No se escribe nada.

### 2. Aplicar

```bash
sm normalize scram.srt --sort --renumber --fix-overlaps
```

```
✓ Subtitles normalized successfully

Backup: backups/scram.srt.2026-06-14_10-36-09.226844
Sorted: yes (by start time)
Renumbered: yes (from 1)
Overlaps fixed: 1
Subtitles: 3
```

El archivo se reescribe en su lugar (se guarda una copia de seguridad en `backups/`
primero). Los fragmentos ahora están ordenados, renumerados desde 1, y el solapamiento
se resuelve recortando el fin del fragmento 1 hasta el inicio del fragmento 2:

```srt
1
00:00:01,000 --> 00:00:03,000
First, overlaps next.

2
00:00:03,000 --> 00:00:05,000
Second.

3
00:00:08,000 --> 00:00:10,000
Third in time.
```

### Valores predeterminados

`--sort` y `--renumber` están **activados por defecto**, así que un simple
`sm normalize scram.srt` ya reordena y renumera — pero deja los solapamientos intactos:

```bash
sm normalize scram.srt --dry-run
```

```
Sorted: yes (by start time)
Renumbered: yes (from 1)
Overlaps fixed: no
```

Añade `--fix-overlaps` solo cuando quieras que los tiempos vecinos sean recortados.

## Salida JSON

```bash
sm --output json normalize scram.srt --sort --renumber --fix-overlaps --dry-run
```

```json
{"ok":true,"data":{"file":"scram.srt","total_count":3,"sorted":true,"renumbered":true,"overlaps_fixed":1,"backup_path":"N/A (dry-run)","dry_run":true}}
```

- `sorted` / `renumbered` — si cada paso se ejecutó.
- `overlaps_fixed` — número de solapamientos vecinos recortados.
- `total_count` — fragmentos tras la normalización.

## `normalize` vs `doctor`

- `sm doctor` **diagnostica y repara problemas estructurales** — códigos de tiempo mal
  formados, bloques de fragmentos rotos, errores de parseo. Úsalo cuando un archivo no
  carga correctamente.
- `sm normalize` asume que el archivo ya es **válido** y lo reordena, renumera y elimina
  solapamientos. Ejecuta `doctor` primero si el parseo falla, luego `normalize` para
  ordenar.

## Problemas y casos extremos

- **`--fix-overlaps` cambia los tiempos**, no solo el orden — recorta el fin de un
  fragmento hasta el inicio del siguiente. Previsualiza con `--dry-run`.
- **Desactiva un valor predeterminado explícitamente** con `--sort=false` /
  `--renumber=false` si quieres renumerar sin reordenar, o viceversa.
- **Renumerar tras una fusión o empalme manual** es el caso común — combina con
  [dedupe](deduplicate-and-clean-after-merge.es.md) cuando también se han colado
  casi-duplicados.

## Véase también

- [Eliminar duplicados y solapamientos tras una fusión](deduplicate-and-clean-after-merge.es.md)
- [Unir partes divididas (CD1/CD2) en un solo archivo](join-split-parts-cd1-cd2.es.md)
- `sm normalize --help`, `sm doctor --help`, `sm describe`
