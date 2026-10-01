# lookalike

Finds photos that are near copies of each other: the same shot saved at a
different size, recompressed by a chat app, lightly edited, or exported
twice. Tools that compare file hashes only catch byte-identical copies and
miss all of these.

Each photo is reduced to a 63-number fingerprint of its overall light and
dark layout. The fingerprints are stored in
[NuclaDB](https://github.com/Rakshit-gen/NuclaDB), and every photo is
searched against the rest to find its look-alikes. Nothing is deleted:
`move` puts the extra copies in another folder and writes an undo log.

## Quickstart

You need Go 1.25 and a running NuclaDB server.

```sh
# NuclaDB (see its README for other install options)
git clone https://github.com/Rakshit-gen/NuclaDB && cd NuclaDB
go build -o bin/nucladbd ./cmd/nucladbd && ./bin/nucladbd -data-dir=./data &

# lookalike
git clone https://github.com/Rakshit-gen/lookalike && cd lookalike
go build -o bin/lookalike ./cmd/lookalike && export PATH=$PWD/bin:$PATH

lookalike scan ~/Pictures/2024
```

```
2 photos, similarity 1.00 to 1.00
  * orig/kodim03.png  (768x512, 491 KB)
    shared/kodim03-small.jpg  (400x266, 9 KB)
...
```

The starred photo in each group is the one `move` keeps: the most pixels,
then the biggest file, then the oldest.

```sh
lookalike report ~/Pictures/2024              # HTML page with the groups side by side
lookalike move ~/Pictures/2024 ~/lookalike-extras          # dry run, prints the plan
lookalike move -apply ~/Pictures/2024 ~/lookalike-extras   # moves, writes an undo log
lookalike undo ~/lookalike-extras/lookalike-undo-20261001-142530.tsv
```

## Commands

| Command | What it does |
|---|---|
| `scan FOLDER` | Fingerprints new or changed photos, updates the index and lists groups. `-min` sets the similarity needed to group (default 0.9), `-rotated` also matches copies turned a quarter or half turn, `-json` prints the groups for scripts. |
| `similar FOLDER PHOTO` | Lists the photos in FOLDER that look like PHOTO, which can be outside the folder. |
| `report FOLDER` | Writes an HTML page with each group's photos side by side. |
| `move FOLDER DEST` | Moves every photo but the keeper in each group to DEST, keeping subfolders. Dry run unless `-apply`. |
| `undo LOGFILE` | Moves files back, newest first. Skips any whose original spot is taken. |
| `forget FOLDER` | Drops the cache and the NuclaDB tenant for FOLDER. Photos are not touched. |
| `eval FOLDER` | Measures grouping on edited copies of the photos in FOLDER. See below. |

JPEG, PNG, GIF and WebP are read. Hidden files and folders are skipped.

Environment: `NUCLADB_URL` (default `http://localhost:8080`),
`NUCLADB_API_KEY`, `LOOKALIKE_CACHE` (default: your user cache directory).

## How well it works

`eval` takes a folder of photos that are all different, writes six edited
copies of each to a temporary folder, indexes them in a temporary NuclaDB
tenant and counts how many copies end up in their original's group. It
also counts wrong merges: pairs of different originals put in one group.

On the 24 photos of the Kodak test set (768x512):

| `-min` | half size | JPEG quality 20 | brighter | 5% crop | 15% crop | quarter turn | wrong merges |
|---|---|---|---|---|---|---|---|
| 0.80 | 100% | 100% | 100% | 100% | 46% | 0% | 9 |
| 0.85 | 100% | 100% | 100% | 96% | 29% | 0% | 0 |
| **0.90 (default)** | **100%** | **100%** | **100%** | **92%** | **8%** | **0%** | **0** |
| 0.95 | 100% | 100% | 100% | 46% | 0% | 0% | 0 |

With `-rotated`, quarter turns are found 100% of the time at every
threshold, and wrong merges at 0.80 go from 9 to 10. The other columns
don't change.

So: resized, recompressed and brightened copies are always found. Small
crops mostly are. A crop that cuts 15% off each side changes the layout
too much and is usually missed. Below 0.85 different photos with similar
composition (two landscapes with the horizon at the same height) start to
merge.

24 photos is a small set. The numbers say the method works on these
edits; they don't say how many false groups a real library of 20,000
holiday photos will show at 0.90. Look at `report` before `move -apply`.

## Measured vs Claimed

| Claim | Value | How measured | Date |
|---|---|---|---|
| Resized, recompressed and brightened copies grouped | 100% (72 of 72) | `lookalike eval` on the 24 Kodak photos, `-min 0.9` | 2026-10-01 |
| 5% crops grouped | 92% (22 of 24) | same | 2026-10-01 |
| Wrong merges at the default threshold | 0 | same | 2026-10-01 |
| `eval` run time | 2.3 s | 168 generated photos, local NuclaDB, Apple M4 | 2026-10-01 |
| First `scan` of 2000 photos | 3.5 s | 1200x900 JPEGs, 10 cores, fingerprinting plus indexing plus grouping | 2026-10-01 |
| Repeat `scan`, nothing changed | 0.23 s | same 2000 photos, 2000 searches | 2026-10-01 |

## How it works

```
photo ──decode──▶ 32x32 grayscale grid (EXIF rotation applied)
                         │
                         ▼
          2D DCT, keep the 8x8 lowest frequencies
          minus the average brightness: 63 numbers
                         │
                         ▼
     NuclaDB tenant per folder, cosine, id = hash of the path
                         │
         scan: search each photo's 10 nearest neighbours,
         link pairs above -min, groups = connected sets
```

- `internal/photo`: decoding, the EXIF orientation tag, the grid, the DCT
  fingerprint, and rotating a fingerprint without touching the pixels
  (a quarter turn of the image is a transpose and some sign flips of the
  DCT coefficients). `-rotated` searches with those.
- `internal/library`: walking the folder and a cache of fingerprints keyed
  by path, size and modification time, so only new or changed photos are
  decoded again.
- `internal/group`: pushing fingerprints to NuclaDB and building groups.
- `internal/mover`: planning and applying moves, the undo log and undo.
  The log is written and synced before each file moves, and a move across
  disks copies then removes.
- `internal/variant`: the edits `eval` makes.
- `internal/nucla`: a small NuclaDB REST client. `nucla/nuclafake` is an
  in-memory stand-in for tests.

Leaving out the average brightness is what makes brighter copies match.
Dropping everything above the 8x8 lowest frequencies is what makes resized
and recompressed copies match, and also why crops of more than about 10%
are missed: they shift the whole layout.

## Status

Works:
- All commands on macOS. Linux should work but has not been tried.
- Unit tests for every package against the in-memory fake. One
  integration test runs against a real server when `NUCLADB_TEST_URL` is
  set.

Known limits:
- No HEIC or RAW. Convert first, or those photos are skipped.
- Mirrored copies are not matched, even with `-rotated`.
- Crops beyond about 10% per side are mostly missed.
- Grouping is single linkage: a long burst of slowly changing shots can
  chain into one group even when the first and last frames differ a lot.
- Tested on the 24 Kodak photos and a generated set. Not yet on a large
  personal library.
