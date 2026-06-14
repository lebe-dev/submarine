# Resincronizar solo una parte del archivo

> El desplazamiento apareció a mitad del archivo — tras un empalme o una escena insertada — así que solo los fragmentos posteriores necesitan moverse.

## Escenario

La primera mitad del archivo está sincronizada, pero en algún punto del medio (frecuentemente
donde se empalman dos fuentes) el tiempo se desvía y permanece así hasta el final.
Desplazar todo el archivo rompería la parte que ya está correcta. Solo quieres mover el
rango afectado.

## Comandos utilizados

- `sm detect-offset` — medir el deslizamiento en la sección desalineada
- `sm delay` ★ — desplazar solo un rango de fragmentos, por índice o por marca de tiempo

## Archivos de ejemplo

`joined.srt` — los fragmentos 1–2 están bien, los fragmentos 3–4 llegaron 500 ms antes
tras un empalme:

```srt
1
00:00:01,000 --> 00:00:03,000
Part one line A.

2
00:00:04,000 --> 00:00:06,000
Part one line B.

3
00:00:10,000 --> 00:00:12,000
Part two line A.

4
00:00:13,000 --> 00:00:15,000
Part two line B.
```

## Paso a paso

### 1. Desplazar por rango de índice

Mueve los fragmentos 3 al 4 en +500 ms, dejando 1–2 sin cambios. Previsualiza primero:

```bash
sm delay --range 3-4 --dry-run joined.srt +500
```

```
Dry-run: time offset would be applied

Offset: 500 ms
Range: 3-4
Subtitles affected: 2

Sample (first 3 subtitles):
  [1] 00:00:01,000 --> 00:00:03,000  =>  00:00:01,000 --> 00:00:03,000
  [2] 00:00:04,000 --> 00:00:06,000  =>  00:00:04,000 --> 00:00:06,000
  [3] 00:00:10,000 --> 00:00:12,000  =>  00:00:10,500 --> 00:00:12,500
```

Solo los fragmentos 3–4 se mueven (`Subtitles affected: 2`); la muestra confirma que 1 y
2 no cambian.

### 2. O desplazar por marca de tiempo

Cuando conoces el momento pero no el índice, usa `--from-timestamp` — cada fragmento en
o después de ese momento se desplaza:

```bash
sm delay --from-timestamp 00:00:09,000 --dry-run joined.srt +500
```

```
Dry-run: time offset would be applied

Offset: 500 ms
From timestamp: 00:00:09,000
Subtitles affected: 2

Sample (first 3 subtitles):
  [1] 00:00:01,000 --> 00:00:03,000  =>  00:00:01,000 --> 00:00:03,000
  [2] 00:00:04,000 --> 00:00:06,000  =>  00:00:04,000 --> 00:00:06,000
  [3] 00:00:10,000 --> 00:00:12,000  =>  00:00:10,500 --> 00:00:12,500
```

### 3. Aplicar

```bash
sm delay --range 3-4 joined.srt +500
```

Elimina `--dry-run` para escribir el cambio; se guarda una copia de seguridad en
`backups/` primero.

> **Consejo para offsets negativos:** coloca los flags **antes** del offset
> (`sm delay --range 3-4 joined.srt -500`). Un `-` al inicio hace que el offset parezca
> un flag, por lo que los flags colocados después serían ignorados.

## Salida JSON

```bash
sm --output json delay --range 3-4 joined.srt +500
```

```json
{"ok":true,"data":{"offset_ms":500,"subtitles_adjusted":2,"backup_path":"backups/joined.srt.2026-06-14_10-34-15.001252","range_start":3,"range_end":4}}
```

- `subtitles_adjusted` — cuántos fragmentos se movieron.
- `range_start` / `range_end` — el rango de índices que fue desplazado (omitido cuando
  se usa `--from-timestamp`).

## Problemas y casos extremos

- **`--range` y `--from-timestamp` son mutuamente excluyentes:**

  ```bash
  sm delay --range 3-4 --from-timestamp 00:00:09,000 joined.srt +500
  ```
  ```
  error: use either --range or --from-timestamp, not both
  ```

- **Sin ninguno de los dos flags, todo el archivo se desplaza** — ese es el caso de
  [offset constante](detect-and-fix-constant-offset.es.md).
- **`--range` es inclusivo** y usa índices de subtítulos (`START-END`), no marcas de
  tiempo.
- **Mide solo en la sección mala.** Ejecuta `sm detect-offset` contra una referencia
  sincronizada usando solo la parte desalineada para que el offset no se diluya con la
  primera mitad correcta.

## Véase también

- [Detectar y corregir un offset de sincronía constante](detect-and-fix-constant-offset.es.md)
- [Unir partes divididas (CD1/CD2) en un solo archivo](join-split-parts-cd1-cd2.es.md)
- `sm delay --help`, `sm detect-offset --help`, `sm describe`
