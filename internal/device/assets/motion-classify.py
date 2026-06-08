#!/usr/bin/env python3
"""Classify motion snapshot JPEGs with YOLOv8 ONNX.

One-shot:  motion-classify.py <model.onnx> <image.jpg> [input_size] [min_conf]
Worker:    motion-classify.py --worker <model.onnx> [input_size] [min_conf]
           Reads image paths on stdin (one per line), writes JSON lines to stdout.
"""
from __future__ import annotations

import json
import os
import sys

import numpy as np
from PIL import Image

os.environ.setdefault("ORT_LOGGING_LEVEL", "3")

try:
    import onnxruntime as ort
except ImportError as exc:  # pragma: no cover
    print(json.dumps({"error": f"onnxruntime not installed: {exc}"}))
    sys.exit(2)

YOLO_CLASSES = [
    "person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck", "boat",
    "traffic light", "fire hydrant", "stop sign", "parking meter", "bench", "bird", "cat", "dog", "horse",
    "sheep", "cow", "elephant", "bear", "zebra", "giraffe", "backpack", "umbrella", "handbag", "tie",
    "suitcase", "frisbee", "skis", "snowboard", "sports ball", "kite", "baseball bat", "baseball glove",
    "skateboard", "surfboard", "tennis racket", "bottle", "wine glass", "cup", "fork", "knife", "spoon",
    "bowl", "banana", "apple", "sandwich", "orange", "broccoli", "carrot", "hot dog", "pizza", "donut",
    "cake", "chair", "couch", "potted plant", "bed", "dining table", "toilet", "tv", "laptop", "mouse",
    "remote", "keyboard", "cell phone", "microwave", "oven", "toaster", "sink", "refrigerator", "book",
    "clock", "vase", "scissors", "teddy bear", "hair drier", "toothbrush",
]

CATALOG = {
    "person": "human",
    **{k: "animal" for k in ("bird", "cat", "dog", "horse", "sheep", "cow", "elephant", "bear", "zebra", "giraffe")},
    **{k: "vehicle" for k in ("bicycle", "car", "motorcycle", "bus", "train", "truck")},
}


def letterbox(img: Image.Image, size: int, color: tuple[int, int, int] = (114, 114, 114)) -> np.ndarray:
    w, h = img.size
    r = min(size / w, size / h)
    nw, nh = int(round(w * r)), int(round(h * r))
    resized = img.resize((nw, nh), Image.BILINEAR)
    canvas = Image.new("RGB", (size, size), color)
    pad_x = (size - nw) // 2
    pad_y = (size - nh) // 2
    canvas.paste(resized, (pad_x, pad_y))
    arr = np.asarray(canvas, dtype=np.float32) / 255.0
    chw = np.transpose(arr, (2, 0, 1))
    return np.expand_dims(chw, 0)


def parse_nms_output(output: np.ndarray, min_conf: float) -> list[dict]:
    if output.ndim == 3:
        output = output[0]
    out: list[dict] = []
    seen: set[str] = set()
    for row in output:
        if len(row) < 6:
            continue
        score = float(row[4])
        if score < min_conf:
            continue
        class_id = int(row[5])
        if class_id < 0 or class_id >= len(YOLO_CLASSES):
            continue
        label = YOLO_CLASSES[class_id]
        cat = CATALOG.get(label, "")
        if not cat or cat in seen:
            continue
        seen.add(cat)
        out.append({"catalog_type": cat, "raw_label": label, "confidence": score})
    return out


def sigmoid(x: np.ndarray) -> np.ndarray:
    return 1.0 / (1.0 + np.exp(-np.clip(x, -50, 50)))


def parse_raw_output(output: np.ndarray, size: int, min_conf: float) -> list[dict]:
    if output.ndim == 3:
        output = output[0]
    if output.shape[0] == 84:
        rows = output.T
    else:
        rows = output
    n = (size // 8) ** 2 + (size // 16) ** 2 + (size // 32) ** 2
    best_per_cat: dict[str, tuple[float, str]] = {}
    for row in rows[:n]:
        scores = sigmoid(row[4:84])
        class_id = int(scores.argmax())
        conf = float(scores[class_id])
        if conf < min_conf:
            continue
        label = YOLO_CLASSES[class_id]
        cat = CATALOG.get(label, "")
        if not cat:
            continue
        if cat not in best_per_cat or conf > best_per_cat[cat][0]:
            best_per_cat[cat] = (conf, label)
    out: list[dict] = []
    for cat, (conf, label) in sorted(best_per_cat.items(), key=lambda x: -x[1][0]):
        out.append({"catalog_type": cat, "raw_label": label, "confidence": conf})
    return out


def parse_output(output: np.ndarray, size: int, min_conf: float) -> list[dict]:
    if output.ndim >= 2 and output.shape[-1] == 6:
        return parse_nms_output(output, min_conf)
    return parse_raw_output(output, size, min_conf)


def model_max_score(output: np.ndarray) -> float:
    if output.ndim == 3:
        output = output[0]
    if output.ndim >= 2 and output.shape[-1] == 6:
        if len(output) == 0:
            return 0.0
        return float(output[:, 4].max())
    if output.shape[0] == 84:
        rows = output.T
    else:
        rows = output
    best = 0.0
    for row in rows:
        if len(row) < 84:
            continue
        scores = sigmoid(row[4:84])
        best = max(best, float(scores.max()))
    return best


def classify_path(session: ort.InferenceSession, input_name: str, image_path: str, input_size: int, min_conf: float) -> tuple[list[dict], float]:
    img = Image.open(image_path).convert("RGB")
    tensor = letterbox(img, input_size)
    raw = np.asarray(session.run(None, {input_name: tensor})[0])
    return parse_output(raw, input_size, min_conf), model_max_score(raw)


def load_session(model_path: str) -> tuple[ort.InferenceSession, str]:
    session = ort.InferenceSession(model_path, providers=["CPUExecutionProvider"])
    return session, session.get_inputs()[0].name


def worker_loop(model_path: str, input_size: int, min_conf: float) -> int:
    session, input_name = load_session(model_path)
    print(json.dumps({"ready": True}), flush=True)
    for line in sys.stdin:
        path = line.strip()
        if not path:
            continue
        try:
            dets, max_score = classify_path(session, input_name, path, input_size, min_conf)
            print(json.dumps({"detections": dets, "max_score": max_score}), flush=True)
        except Exception as exc:  # noqa: BLE001
            print(json.dumps({"error": str(exc)}), flush=True)
    return 0


def main() -> int:
    if len(sys.argv) >= 2 and sys.argv[1] == "--worker":
        if len(sys.argv) < 3:
            print("usage: motion-classify.py --worker <model.onnx> [input_size] [min_conf]", file=sys.stderr)
            return 1
        model_path = sys.argv[2]
        input_size = int(sys.argv[3]) if len(sys.argv) > 3 else 640
        min_conf = float(sys.argv[4]) if len(sys.argv) > 4 else 0.4
        return worker_loop(model_path, input_size, min_conf)

    if len(sys.argv) < 3:
        print("usage: motion-classify.py <model.onnx> <image.jpg> [input_size] [min_conf]", file=sys.stderr)
        return 1
    model_path = sys.argv[1]
    image_path = sys.argv[2]
    input_size = int(sys.argv[3]) if len(sys.argv) > 3 else 640
    min_conf = float(sys.argv[4]) if len(sys.argv) > 4 else 0.4
    session, input_name = load_session(model_path)
    dets, max_score = classify_path(session, input_name, image_path, input_size, min_conf)
    print(json.dumps({"detections": dets, "max_score": max_score}))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
