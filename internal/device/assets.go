package device

import "embed"

//go:embed assets/motion-classify.py assets/yolov8n.onnx
var runtimeAssets embed.FS
