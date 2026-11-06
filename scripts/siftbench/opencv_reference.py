#!/usr/bin/env python3
"""Offline reference matcher: OpenCV SIFT + BFMatcher + similarity RANSAC.

Same CLI contract as clicker/cmd/siftbench so outputs can be diffed:

    opencv_reference.py --ref a.png --screen b.png [--truth x,y,w,h] [--all]

This is a benchmark aid only. Vibium does not depend on OpenCV. The script
has not been run against an installed cv2 in this checkout; it is byte-compiled
only. Treat its numbers as a sanity check, not as ground truth.
"""

import argparse
import json
import math
import sys
import time

try:
    import cv2
    import numpy as np
except ImportError:  # pragma: no cover - reference only
    print(json.dumps({"matcher": "opencv-sift", "matches": [],
                      "error": "cv2 and numpy are required"}), file=sys.stdout)
    sys.exit(1)

MIN_SCALE, MAX_SCALE = 0.2, 5.0
REPROJ_PX = 4.0
MIN_VISIBLE = 0.5
MAX_OVERLAP_IOU = 0.5
MIN_SPREAD = 0.25


def parse_box(text):
    parts = [int(p) for p in text.split(",")]
    if len(parts) != 4:
        raise ValueError("truth must be x,y,w,h")
    return parts


def iou(a, b):
    ax0, ay0, aw, ah = a
    bx0, by0, bw, bh = b
    ix0, iy0 = max(ax0, bx0), max(ay0, by0)
    ix1, iy1 = min(ax0 + aw, bx0 + bw), min(ay0 + ah, by0 + bh)
    if ix1 <= ix0 or iy1 <= iy0:
        return 0.0
    inter = (ix1 - ix0) * (iy1 - iy0)
    return inter / float(aw * ah + bw * bh - inter)


def knn_ratio_matches(des_ref, des_scr, ratio, k):
    """Generalised Lowe test: keep the shortest prefix of the k nearest
    neighbours that is separated from the next one by the ratio."""
    bf = cv2.BFMatcher(cv2.NORM_L2)
    k = max(2, min(k, len(des_scr)))
    out = []
    for cands in bf.knnMatch(des_ref, des_scr, k=k):
        cands = sorted(cands, key=lambda m: m.distance)
        for m in range(1, len(cands)):
            if cands[m - 1].distance < ratio * cands[m].distance:
                out.extend(cands[:m])
                break
    return out


def project_box(A, w, h):
    pts = np.float32([[0, 0], [w, 0], [0, h], [w, h]]).reshape(-1, 1, 2)
    proj = cv2.transform(pts, A).reshape(-1, 2)
    x0, y0 = proj.min(axis=0)
    x1, y1 = proj.max(axis=0)
    return [int(round(x0)), int(round(y0)), int(round(x1 - x0)), int(round(y1 - y0))]


def find_instances(kp_ref, kp_scr, matches, ref_shape, scr_shape, min_inliers, max_instances):
    ref_h, ref_w = ref_shape[:2]
    scr_h, scr_w = scr_shape[:2]
    pool = list(matches)
    found = []
    ref_pts_all = np.float32([k.pt for k in kp_ref])
    ext_x = max(1.0, float(ref_pts_all[:, 0].max() - ref_pts_all[:, 0].min()))
    ext_y = max(1.0, float(ref_pts_all[:, 1].max() - ref_pts_all[:, 1].min()))
    rounds = 0
    while len(pool) >= min_inliers and len(found) < max_instances and rounds < 2 * max_instances + 4:
        rounds += 1
        src = np.float32([kp_ref[m.queryIdx].pt for m in pool]).reshape(-1, 1, 2)
        dst = np.float32([kp_scr[m.trainIdx].pt for m in pool]).reshape(-1, 1, 2)
        A, mask = cv2.estimateAffinePartial2D(src, dst, method=cv2.RANSAC,
                                              ransacReprojThreshold=REPROJ_PX,
                                              maxIters=2000, confidence=0.99)
        if A is None or mask is None:
            break
        inl = [i for i, v in enumerate(mask.ravel()) if v]
        if len(inl) < min_inliers:
            break
        a, b = float(A[0, 0]), float(A[1, 0])
        scale = math.hypot(a, b)
        rotation = math.atan2(b, a)
        box = project_box(A, ref_w, ref_h)
        ok = MIN_SCALE <= scale <= MAX_SCALE and box[2] >= 4 and box[3] >= 4
        if ok:
            vis = iou_visible(box, scr_w, scr_h)
            ok = vis >= MIN_VISIBLE
        if ok:
            rp = np.float32([kp_ref[pool[i].queryIdx].pt for i in inl])
            sx = (rp[:, 0].max() - rp[:, 0].min()) / ext_x
            sy = (rp[:, 1].max() - rp[:, 1].min()) / ext_y
            ok = sx >= MIN_SPREAD and sy >= MIN_SPREAD
        if ok:
            ok = all(iou(f["box"], box) <= MAX_OVERLAP_IOU for f in found)
        drop = set(inl)
        if ok:
            x, y, w, h = box
            for i, m in enumerate(pool):
                px, py = kp_scr[m.trainIdx].pt
                if x <= px < x + w and y <= py < y + h:
                    drop.add(i)
            found.append({
                "box": box,
                "inliers": len(inl),
                "matched": len(drop),
                "scale": scale,
                "rotation": rotation,
            })
        pool = [m for i, m in enumerate(pool) if i not in drop]
    return found


def iou_visible(box, scr_w, scr_h):
    x, y, w, h = box
    vx0, vy0 = max(x, 0), max(y, 0)
    vx1, vy1 = min(x + w, scr_w), min(y + h, scr_h)
    if vx1 <= vx0 or vy1 <= vy0 or w * h <= 0:
        return 0.0
    return (vx1 - vx0) * (vy1 - vy0) / float(w * h)


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--ref", required=True)
    ap.add_argument("--screen", required=True)
    ap.add_argument("--truth", default="")
    ap.add_argument("--all", action="store_true")
    ap.add_argument("--ratio", type=float, default=0.75)
    ap.add_argument("--min-inliers", type=int, default=8)
    ap.add_argument("--max-instances", type=int, default=10)
    args = ap.parse_args()

    ref = cv2.imread(args.ref, cv2.IMREAD_GRAYSCALE)
    scr = cv2.imread(args.screen, cv2.IMREAD_GRAYSCALE)
    if ref is None or scr is None:
        print(json.dumps({"matcher": "opencv-sift", "matches": [], "error": "cannot read input"}))
        return 1

    start = time.perf_counter()
    sift = cv2.SIFT_create(contrastThreshold=0.04, edgeThreshold=10, sigma=1.6)
    kp_ref, des_ref = sift.detectAndCompute(ref, None)
    kp_scr, des_scr = sift.detectAndCompute(scr, None)
    out = {
        "matcher": "opencv-sift",
        "matches": [],
        "stats": {
            "ref_keypoints": len(kp_ref),
            "screen_keypoints": len(kp_scr),
            "ratio_matches": 0,
            "elapsed_ms": 0.0,
        },
    }
    if des_ref is None or len(kp_ref) < args.min_inliers:
        out["error"] = "reference image is too small or flat to match"
    elif des_scr is not None and len(kp_scr) > 0:
        matches = knn_ratio_matches(des_ref, des_scr, args.ratio, args.max_instances + 1)
        out["stats"]["ratio_matches"] = len(matches)
        found = find_instances(kp_ref, kp_scr, matches, ref.shape, scr.shape,
                               args.min_inliers, args.max_instances)
        found.sort(key=lambda f: (-f["inliers"], f["box"][1], f["box"][0]))
        if not args.all:
            found = found[:1]
        for f in found:
            x, y, w, h = f["box"]
            vx0, vy0 = max(x, 0), max(y, 0)
            vx1, vy1 = min(x + w, scr.shape[1]), min(y + h, scr.shape[0])
            clipped = [vx0, vy0, vx1 - vx0, vy1 - vy0]
            out["matches"].append({
                "box": {"X": clipped[0], "Y": clipped[1], "W": clipped[2], "H": clipped[3]},
                "center": [(vx0 + vx1) // 2, (vy0 + vy1) // 2],
                "inliers": f["inliers"],
                "matched": f["matched"],
                "scale": f["scale"],
                "rotation": f["rotation"],
                "score": f["inliers"] / float(len(kp_ref)),
            })
    out["stats"]["elapsed_ms"] = (time.perf_counter() - start) * 1000

    if args.truth:
        tb = parse_box(args.truth)
        best = 0.0
        for m in out["matches"]:
            b = m["box"]
            best = max(best, iou([b["X"], b["Y"], b["W"], b["H"]], tb))
        out["iou_vs_truth"] = best

    json.dump(out, sys.stdout, indent=2)
    sys.stdout.write("\n")
    return 1 if "error" in out else 0


if __name__ == "__main__":
    sys.exit(main())
