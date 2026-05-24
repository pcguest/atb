#!/usr/bin/env bash
# Record launch demo assets from the tamper-demo workflow.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ASSETS="$ROOT/docs/launch/assets"
TMP="$(mktemp -d)"
FONT="/System/Library/Fonts/Menlo.ttc"
if [[ ! -f "$FONT" ]]; then
  FONT="/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
fi

ATB="$ROOT/atb"
if [[ ! -x "$ATB" ]]; then
  (cd "$ROOT" && go build -o "$ATB" ./cmd/atb)
fi

VENV="$TMP/venv"
python3 -m venv "$VENV"
"$VENV/bin/pip" -q install pillow

cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

cd "$TMP"

run_step() {
  shift # optional label (display only)
  {
    echo "\$ $*"
    "$@"
  } >"$TMP/step-$(printf '%02d' "$STEP").txt" 2>&1 || true
  STEP=$((STEP + 1))
}

render_text_png() {
  local textfile=$1
  local out=$2
  local fontsize=${3:-18}
  "$VENV/bin/python3" - "$textfile" "$out" "$FONT" "$fontsize" <<'PY'
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

textfile, out, font_path, fontsize = sys.argv[1:5]
fontsize = int(fontsize)
text = Path(textfile).read_text()
lines = text.splitlines() or [""]

width, height = 1280, 720
bg = (0x0D, 0x11, 0x17)
fg = (0xE6, 0xED, 0xF3)
img = Image.new("RGB", (width, height), bg)
draw = ImageDraw.Draw(img)
try:
    font = ImageFont.truetype(font_path, fontsize)
except OSError:
    font = ImageFont.load_default()

x, y = 24, 24
line_spacing = 8
for line in lines:
    draw.text((x, y), line, font=font, fill=fg)
    bbox = draw.textbbox((x, y), line, font=font)
    y = bbox[3] + line_spacing

img.save(out)
PY
}

STEP=1
run_step "bundle new" "$ATB" bundle new
run_step "append request" "$ATB" append ai.request.received \
  --data '{"request_id":"req-demo","actor_id_hash":"sha256-user-demo","purpose_tag":"tamper-demo"}'
run_step "append policy" "$ATB" append ai.policy.decision \
  --data '{"policy_id":"pol-demo","policy_version":"1","decision":"allow","decision_reason_codes":["demo"],"subject_id_hash":"sha256-user-demo","action_id":"act-demo"}'
run_step "verify pass" "$ATB" verify --profile atb.profile.policy_decision --format json

python3 - <<'PY'
from pathlib import Path
p = Path("run.atb/bundle.atb")
data = bytearray(p.read_bytes())
for i, b in enumerate(data):
    if b == ord("d") and i > 100:
        data[i] = ord("X")
        break
p.write_bytes(data)
PY

run_step "verify fail" "$ATB" verify --profile atb.profile.policy_decision --format json

mkdir -p "$TMP/frames"
frame_idx=0
for file in "$TMP"/step-*.txt; do
  [[ -f "$file" ]] || continue
  frame_idx=$((frame_idx + 1))
  out="$TMP/frames/frame-$(printf '%03d' "$frame_idx").png"
  render_text_png "$file" "$out" 18
done

mkdir -p "$ASSETS"
ffmpeg -y -loglevel error -framerate 1 -i "$TMP/frames/frame-%03d.png" \
  -vf "fps=1,scale=960:-1:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse" \
  "$ASSETS/atb-verify-demo.gif"

"$ATB" verify --profile atb.profile.policy_decision --format json \
  >"$TMP/verify-report.json" 2>/dev/null || true

"$VENV/bin/python3" - <<'PY' "$TMP/verify-report.json" "$ASSETS/atb-verify-report.png" "$FONT"
import json, sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

report_path, out_png, font_path = sys.argv[1:4]
report = json.loads(Path(report_path).read_text())
lines = [
    "ATB verify report (policy_decision)",
    f"pass: {report.get('pass')}",
    f"profile: {report.get('profile_id', 'atb.profile.policy_decision')}",
]
integrity = report.get("integrity") or {}
if integrity:
    lines.append(f"chain_valid: {integrity.get('chain_valid')}")
cas = report.get("cas") or {}
if cas:
    lines.append(f"cas_grade: {cas.get('grade', 'n/a')}")
    lines.append(f"cas_overall: {cas.get('overall', 'n/a')}")
gaps = report.get("provability_gaps") or []
lines.append(f"provability_gaps: {len(gaps)}")
for gap in gaps[:4]:
    lines.append(f"  [{gap.get('layer')}] {gap.get('gap')}")
failures = report.get("critical_failures") or []
if failures:
    lines.append("critical_failures:")
    for f in failures[:3]:
        lines.append(f"  {f.get('kind')}: {f.get('detail')}")

width, height = 1280, 720
bg = (0x0D, 0x11, 0x17)
fg = (0xE6, 0xED, 0xF3)
img = Image.new("RGB", (width, height), bg)
draw = ImageDraw.Draw(img)
try:
    font = ImageFont.truetype(font_path, 20)
except OSError:
    font = ImageFont.load_default()
x, y = 32, 32
for line in lines:
    draw.text((x, y), line, font=font, fill=fg)
    bbox = draw.textbbox((x, y), line, font=font)
    y = bbox[3] + 10
img.save(out_png)
PY

echo "Wrote $ASSETS/atb-verify-demo.gif"
echo "Wrote $ASSETS/atb-verify-report.png"
