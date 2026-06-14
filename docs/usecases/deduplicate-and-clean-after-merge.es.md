# Eliminar duplicados y solapamientos tras una fusión

> Tras una fusión o empalme agresivo, líneas casi idénticas se acumulan casi al mismo tiempo — colápssalas y luego ordena el archivo.

## Escenario

La combinación de pistas dejó el archivo con casi-duplicados: la misma línea aparece dos
veces con pocos cientos de milisegundos de diferencia, a veces una vez con formato `<i>` y
otra sin él. `sm dedupe` colapsa los fragmentos que comparten el mismo texto (normalizado)
y se solapan en el tiempo, conservando un único fragmento fusionado; `sm normalize` luego
reordena y renumera.

## Comandos utilizados

- `sm dedupe` ★ — colapsar fragmentos casi duplicados
- `sm normalize` — ordenación y renumeración final

## Archivos de ejemplo

`dup.srt` — cada línea aparece dos veces, con ~50–100 ms de diferencia, una copia con HTML:

```srt
1
00:00:01,000 --> 00:00:03,000
<i>Hello there.</i>

2
00:00:01,100 --> 00:00:03,100
Hello there.

3
00:00:05,000 --> 00:00:07,000
A unique line.

4
00:00:05,050 --> 00:00:07,050
A unique line.
```

## Paso a paso

### 1. Previsualizar con `--dry-run`

`--time-tolerance` es la distancia máxima (en ms) entre dos fragmentos con el mismo texto
para que aún se consideren la misma línea; `--ignore-html` elimina las etiquetas antes de
comparar, por lo que `<i>Hello there.</i>` y `Hello there.` coinciden:

```bash
sm dedupe dup.srt --time-tolerance 200 --ignore-html --dry-run
```

```
Dry-run: duplicates would be removed

Time tolerance: 200 ms
Original subtitles: 4
Removed: 2
Merged groups: 2
Final subtitles: 2
```

Dos pares colapsan en dos fragmentos: `Removed: 2`, `Merged groups: 2`, `Final subtitles: 2`.

### 2. Aplicar

```bash
sm dedupe dup.srt --time-tolerance 200 --ignore-html
```

```
✓ Duplicates removed successfully

Backup: backups/dup.srt.2026-06-14_10-36-09.200473
Original subtitles: 4
Removed: 2
Merged groups: 2
Final subtitles: 2
```

Cada fragmento fusionado conserva el texto de la primera ocurrencia y abarca la unión de
las dos ventanas (inicio más temprano, fin más tardío):

```srt
1
00:00:01,000 --> 00:00:03,100
<i>Hello there.</i>

2
00:00:05,000 --> 00:00:07,050
A unique line.
```

### 3. Ordenar el resultado

Renumera y reordena el archivo limpio:

```bash
sm normalize dup.srt
```

Consulta [Normalizar estructura](normalize-structure.es.md) para más detalles.

## Salida JSON

```bash
sm --output json dedupe dup.srt --time-tolerance 200 --ignore-html --dry-run
```

```json
{"ok":true,"data":{"file":"dup.srt","original_count":4,"removed":2,"merged":2,"final_count":2,"time_tolerance_ms":200,"backup_path":"N/A (dry-run)","dry_run":true}}
```

- `original_count` / `final_count` — número de fragmentos antes y después.
- `removed` — fragmentos eliminados como duplicados.
- `merged` — número de grupos colapsados.

## Problemas y casos extremos

- **`--time-tolerance 0` requiere ventanas que se solapen** para fusionar — solo los
  fragmentos que realmente se intersectan en tiempo y comparten texto colapsan. Amplía la
  tolerancia para detectar copias que estén apenas separadas.
- **Sin `--ignore-html`, las diferencias de formato mantienen los fragmentos separados** —
  `<i>Hello</i>` y `Hello` se tratan como texto distinto.
- **El texto del primer fragmento prevalece** en un grupo fusionado, por lo que una copia
  en cursiva puede sobrevivir sobre una sin formato (o viceversa) según el orden. Ejecuta
  [normalize](normalize-structure.es.md) después para renumerar.

## Véase también

- [Normalizar la estructura: ordenar, renumerar, corregir solapamientos](normalize-structure.es.md)
- [Fusionar una traducción incompleta con un archivo donante](merge-incomplete-translation-with-donor.es.md)
- `sm dedupe --help`, `sm normalize --help`, `sm describe`
