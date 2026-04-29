#!/usr/bin/env bash
set -e
BASE=/mnt/c/Users/Claudio/Documents/Libraries/glfw-purego

for VER in 3.0 3.1 3.2 3.3 3.4; do
  DIR=/tmp/smoke_v${VER}
  rm -rf "$DIR"
  mkdir -p "$DIR"

  # v3.3 lives in the ROOT module; all others have their own go.mod
  if [ "$VER" = "3.3" ]; then
    cat > "$DIR/go.mod" <<EOF
module smoke_v${VER}
go 1.25.0
require github.com/ClaudioTheobaldo/glfw-purego v0.0.0-00010101000000-000000000000
replace github.com/ClaudioTheobaldo/glfw-purego => ${BASE}
EOF
  else
    cat > "$DIR/go.mod" <<EOF
module smoke_v${VER}
go 1.25.0
require github.com/ClaudioTheobaldo/glfw-purego/v${VER}/glfw v0.0.0-00010101000000-000000000000
replace (
  github.com/ClaudioTheobaldo/glfw-purego/v${VER}/glfw => ${BASE}/v${VER}/glfw
  github.com/ClaudioTheobaldo/glfw-purego => ${BASE}
)
EOF
  fi

  cat > "$DIR/main.go" <<'GOEOF'
package main

import (
  "fmt"
  "os"
  glfw "IMPORT_PATH"
)

func main() {
  if err := glfw.Init(); err != nil {
    fmt.Fprintf(os.Stderr, "Init: %v\n", err)
    os.Exit(1)
  }
  defer glfw.Terminate()

  glfw.WindowHint(glfw.ContextVersionMajor, 2)
  glfw.WindowHint(glfw.ContextVersionMinor, 0)

  w, err := glfw.CreateWindow(320, 240, "smoke VVER", nil, nil)
  if err != nil {
    fmt.Fprintf(os.Stderr, "CreateWindow: %v\n", err)
    os.Exit(1)
  }
  defer w.Destroy()

  w.MakeContextCurrent()
  glfw.SwapInterval(1)

  width, height := w.GetSize()
  fw, fh := w.GetFramebufferSize()

  fmt.Printf("v VVER  window=%dx%d  fb=%dx%d  time=%.3fs\n",
    width, height, fw, fh, glfw.GetTime())
}
GOEOF

  if [ "$VER" = "3.3" ]; then
    sed -i "s|IMPORT_PATH|github.com/ClaudioTheobaldo/glfw-purego/v3.3/glfw|g" "$DIR/main.go"
  else
    sed -i "s|IMPORT_PATH|github.com/ClaudioTheobaldo/glfw-purego/v${VER}/glfw|g" "$DIR/main.go"
  fi
  sed -i "s|VVER|${VER}|g" "$DIR/main.go"

  echo -n "building v${VER}... "
  (cd "$DIR" && go mod tidy 2>/dev/null && go build -o /tmp/smoke_v${VER}_bin .) \
    && echo "OK" || echo "FAIL"
done

echo ""
echo "--- running ---"
for VER in 3.0 3.1 3.2 3.3 3.4; do
  DISPLAY=:0 /tmp/smoke_v${VER}_bin 2>/dev/null
done
