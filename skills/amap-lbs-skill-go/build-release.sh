#!/usr/bin/env bash
# 正式发布：交叉编译 4 个 CLI，打成 linux/amd64 与 linux/arm64 两个发布包 + SHA256SUMS。
# 用法：./build-release.sh          （默认 linux amd64+arm64）
#       PLATFORMS="linux/amd64 linux/arm64 darwin/arm64" ./build-release.sh
set -euo pipefail
cd "$(dirname "$0")"

NAME=amap-lbs-skill-go
CMDS=(geocode poi-search route-planning travel-planner)
PLATFORMS=${PLATFORMS:-"linux/amd64 linux/arm64"}
OUT=dist

VERSION=$(grep -m1 '^version:' SKILL.md | awk '{print $2}')
[ -n "$VERSION" ] || { echo "无法从 SKILL.md 解析 version"; exit 1; }

rm -rf "$OUT"; mkdir -p "$OUT"
echo "发布 $NAME v$VERSION → $OUT/"

for plat in $PLATFORMS; do
  os=${plat%/*}; arch=${plat#*/}
  pkg="${NAME}_${VERSION}_${os}_${arch}"
  stage="$OUT/$pkg"
  mkdir -p "$stage/bin"

  for cmd in "${CMDS[@]}"; do
    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
      go build -trimpath -ldflags="-s -w" -o "$stage/bin/$cmd" "./cmd/$cmd"
  done

  # 随包文档/示例；SKILL.md 转为「预编译二进制」版本
  cp README.md config.example.json LICENSE "$stage/"
  python3 - "$stage/SKILL.md" <<'PY'
import sys
src = open("SKILL.md", encoding="utf-8").read()
# 命令示例：源码 go run → 预编译二进制
src = src.replace("go run ./cmd/", "./bin/")
# 去掉依赖源码的 `go build ./...`（二进制包无源码）
src = "\n".join(l for l in src.split("\n") if not l.startswith("go build ./..."))
# frontmatter：不再依赖 go，也无需 install
src = src.replace("      bins:\n        - go\n", "      bins: []\n")
src = src.replace('    install:\n      - kind: go\n        package: ""\n        bins: []\n', "    install: []\n")
open(sys.argv[1], "w", encoding="utf-8").write(src)
PY

  tar -C "$OUT" -czf "$OUT/$pkg.tar.gz" "$pkg"
  rm -rf "$stage"
  echo "  ✓ $OUT/$pkg.tar.gz"
done

( cd "$OUT" && shasum -a 256 ./*.tar.gz > SHA256SUMS )
echo "校验和：$OUT/SHA256SUMS"
