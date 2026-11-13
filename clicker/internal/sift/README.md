# sift

Pure-Go SIFT feature matcher. It locates every occurrence of a small
reference image (icon, thumbnail, crop) inside a screenshot and returns a
verified box, scale and rotation per instance. It powers `vibium find image`.

Standard library only. No CGO, no OpenCV, no template matching. Builds with
`CGO_ENABLED=0` for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 and
windows/amd64.

## API

```go
matches, stats, err := sift.FindAll(ref, screen, sift.DefaultOptions())
best, stats, err := sift.Find(ref, screen, sift.DefaultOptions()) // nil, nil when absent
```

`ErrTooSmall` (reference under 8×8), `ErrTooLarge` (over `MaxPixels`) and
`ErrNoFeatures` (reference yields fewer keypoints than `MinInliers`, so it can
never be verified) are errors. "Nothing found" is an empty slice with a nil
error. A screen with no features is not an error. No match is ever returned
without geometric verification.

`Match.Score` is `Inliers / reference keypoints`. It is raw support, not a
probability.

## Pipeline and thresholds

| Stage | Setting | Why |
| --- | --- | --- |
| Gray | Rec. 601 luminance, [0,1]; alpha composited over mid-gray | Neutral guess for an unknown page background |
| Contrast | Global stretch of the 0.5th–99.5th percentiles | Measured best; see below |
| Reference padding | 24 px edge-replicated; keypoints in the pad are dropped | A crop's surroundings are unknown; mirroring would invent structure |
| Blur borders | Edge-replicate | Same reason; applies to both images |
| Pyramid | σ0 = 1.6, camera blur 0.5, 3 intervals, octaves until min side < 16 (max 8), 4σ kernels | Lowe 2004 |
| Upsample | Reference when its small side ≤ 256 px; screen when ≤ 8.5 Mpx | Lowe's −1 octave; see "Screen upsampling" |
| Extrema | 3×3×3, prefilter ½·0.04/3, 5-px border | Lowe / standard |
| Refinement | Quadratic, up to 5 moves, reject when \|D̂\|·3 < 0.04 | Lowe sec. 4 |
| Edges | Reject when tr²/det ≥ (r+1)²/r, r = 10 | Lowe sec. 4.1 |
| Orientation | 36 bins, 1.5σ Gaussian window, [1 4 6 4 1]/16 smoothing, peaks ≥ 80 % | Lowe sec. 5 |
| Descriptor | 4×4×8, 3σ bins, trilinear, σw = 2 bins, normalise, clamp 0.2, renormalise | Lowe sec. 6 |
| Cap | Strongest `MaxFeatures` (10000) by \|D̂\|, total tie-break | Deterministic |
| Matching | Brute-force k-NN, k = MaxInstances+1 (≤ 11), \|a\|²+\|b\|²−2a·b, parallel rows | Fast enough; no k-d tree needed |
| Ratio | 0.75, chain rule (below) | Lowe, extended to repeated instances |
| RANSAC | 2-point similarity, seeded PCG, ≤ 2000 iterations, adaptive 99 %, ≥ 100 | Bounded and deterministic |
| Inlier | Reprojection ≤ 4 px × max(1, scale); keypoint scale ratio within 2× and orientation within 45° of the model | Cheap extra verification |
| Accept | Inliers ≥ `MinInliers` (8); scale ∈ [0.2, 5]; box ≥ 4 px per side, ≥ 50 % on screen; inlier spread ≥ 25 % of the reference's keypoint extent on both axes; IoU with an accepted instance ≤ 0.5 | Rejects tiny-patch and degenerate fits |
| Multi-instance | Remove inliers and matches inside the box; repeat; ≤ 2·MaxInstances+4 rounds | Spec |
| Output | Sorted by Inliers desc, Box.Min.Y, Box.Min.X; `Box` clipped to the screen; `Center` of the clipped box | Deterministic, clickable |

### Ratio chain rule

Lowe's test compares the nearest neighbour with the second nearest. With two
identical copies on screen both are equally close and every match dies. With
an exact copy and a resampled copy the exact one (d ≈ 0) makes any second
distance look like a gap, so the second copy never matches.

The matcher takes the k nearest neighbours, finds the first `j` with
`d[j] < 0.75·d[j+1]`, then keeps extending while the next neighbour is itself
distinctive against its successor. The accepted prefix is `1..j`. With k = 2
this is exactly Lowe's test. More than `MaxInstances` identical copies still
defeat it (all k distances tie); that is a known limitation.

### Screen upsampling

Doubling the screen before its pyramid (Lowe's −1 octave) tripled the
inliers on the 1× fixtures (34 → 101 on the exact copy) and turned the 80-px
icon fixture from 0 matches into 47 inliers: small flat icons live almost
entirely at σ < 1.8 px, which a native-resolution pyramid never searches.
Screens above 8.5 Mpx are left at native resolution: they are either high-DPI
captures, which already hold those scales, or so large that the doubled
octave dominates latency.

Octaves are processed in horizontal bands of ≤ 2 Mpx with a 100-row overlap
(summed blur radii 43 + widest descriptor window 38 + refinement walk 16), so
the banded result equals the unbanded one and the doubled screen is never
resident at full size. Band layout depends only on image size, so results
are identical for any GOMAXPROCS (checked at 1, 3, 4 and 12, in-process and
across processes).

### Contrast normalisation (measured)

Copies with contrast α and brightness β (`v' = α(v−128)+128+255β`) pasted on
the 1280×800 fixture screen. `inl` is inliers of the found instance, 0 means
not found. `SIFT_REPORT=1 go test -run ReportNormalization` reproduces it.

| α / β | none | global stretch | local (σ 24, floor 0.1) |
| --- | --- | --- | --- |
| 1.0 / 0 | 104 | 104 | 88 |
| 0.6 / +0.15 | 98 | 99 | 90 |
| 0.4 / +0.20 | 74 | 82 | 74 |
| 0.3 / +0.30 | 47 | 51 | 36 |
| 0.2 / +0.30 | 23 | 35 | 25 |
| 0.15 / +0.35 | 0 | 11 | 11 |
| lookalike false positives | 0 | 0 | 0 |

Global stretch wins or ties at every level and keeps the lookalike guard
clean, so it is the default. Local normalisation lifted the weakest case but
cost 15 % of the inliers on unmodified copies and ~25 % latency.

## Measured numbers

macOS VM guest (12 vCPUs, Apple M4 Max host), Go 1.26, `DefaultOptions`,
96×96 reference, 20 runs. `SIFT_REPORT=1 go test -run Report -v` reproduces.
Timings on this VM are indicative; two runs are shown.

| Screen | p50 | p95 | ref / screen keypoints | ratio matches | alloc per run | peak heap in use (GOGC=100) |
| --- | --- | --- | --- | --- | --- | --- |
| 1280×800 | 82–90 ms | 86–106 ms | 122 / 2079 | 104 | 137 MB | 129 MB |
| 2560×1600 | 308–337 ms | 329–364 ms | 122 / 7351 | 73 | 371 MB | 349 MB |

Where the time goes (1280×800): Gaussian pyramid + DoG ≈ 60 ms, descriptors
≈ 10 ms, extrema ≈ 6 ms, matching ≈ 1 ms, RANSAC < 0.1 ms. The blur runs at
2.9–3.7 GMAC/s per core and scales ~8.5× across 12 vCPUs.

Fixture results (procedural, anti-aliased, 1280×800 screen with gradient,
noise, three decoy icons and text blocks):

| Case | Result |
| --- | --- |
| Exact copy | 1 match, IoU 1.00, centre error 0 px, 101 inliers |
| Copies at 0.5× and 1.5× | both found, scale 0.501 / 1.501, IoU 1.00, 24 / 70 inliers |
| α 0.6, β +0.15 | found, 99 inliers |
| 25 % occluded | found, IoU 1.00, 66 inliers |
| Three copies | exactly 3, disjoint, each IoU ≥ 0.8 |
| Same-style lookalike, reference absent | 0 matches (1 ratio survivor) |
| 8×8 and solid references | `ErrNoFeatures`; 7×7 is `ErrTooSmall` |
| Reference absent | 0 matches |
| High-DPI (screen and copy at 2×) | found, scale 2.003, IoU 0.99, 70 inliers |
| PNG fixtures: 80-px icon at 1× and 1.25× in a mock window | both found, IoU 1.00, 47 / 37 inliers; absent icon 0 matches |

Binary size: a throwaway main that decodes a PNG and calls `FindAll` versus
the same main without the call. darwin/arm64 2,691,074 → 2,869,730 bytes
(+175 KB); linux/amd64 `-s -w` 1,736,866 → 1,876,130 bytes (+136 KB). The
package's own symbols are 36 KB; the rest is standard library pulled in.
Against the 17.7 MB `vibium` binary this is about +1 %.

## Harness

```
go run ./cmd/siftbench --ref icon.png --screen shot.png [--truth x,y,w,h] [--all]
python3 scripts/siftbench/opencv_reference.py --ref icon.png --screen shot.png [--truth x,y,w,h] [--all]
```

Both print the same JSON shape (`matches[].box/inliers/scale/score`,
`stats`, `iou_vs_truth`). The Python script needs `cv2` and `numpy`, which
are not installed here; it is byte-compiled but unverified against OpenCV.

Regenerate `testdata/` with `SIFT_GEN_FIXTURES=1 go test -run GenerateFixtures`;
`TestFixturesInSync` fails when the PNGs drift from the generator.

## Known limitations

- Flat or tiny targets yield too few keypoints. Solid icons, thin glyphs and
  references under ~32 px usually return `ErrNoFeatures` or no match.
- Rotation is handled by orientation assignment but is untested beyond a few
  degrees; the 45° orientation gate is relative to the fitted model, so
  rotated instances are not rejected by design.
- The transform is a similarity. Mirrored, sheared or perspective instances
  are not found.
- More than `MaxInstances` identical copies defeat the ratio test.
- A reference whose only distinctive content is a generic sub-element (a
  checkbox, a chevron) fails the 25 % spread test on purpose.
- Screens above 8.5 Mpx skip the doubled octave, so small 1× icons on very
  large screenshots have lower recall.
- Peak memory scales with band area (≈ 10 layers × 2 Mpx × 4 B plus bases);
  lower `bandPixels` to trade time for memory.

## License note

This is an original Apache-2.0 implementation written from Lowe (2004), with
no third-party code or weights.
