#!/usr/bin/env bash
set -euo pipefail

echo "Pinning GitHub Actions to SHA for v1.1.0..."

PINS=$(cat <<'EOF'
actions/checkout@v4 actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5
actions/download-artifact@v4 actions/download-artifact@d3f86a106a0bac45b974a628896c90dbdf5c8093
actions/github-script@v7 actions/github-script@f28e40c7f34bde8b3046d885e986cb6290c5673b
actions/labeler@v5 actions/labeler@8558fd74291d67161a8a78ce36a881fa63b766a9
actions/setup-go@v5 actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff
actions/setup-node@v4 actions/setup-node@49933ea5288caeca8642d1e84afbd3f7d6820020
actions/setup-python@v5 actions/setup-python@a26af69be951a213d495a4c3e4e4022e16d87065
actions/upload-artifact@v4 actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02
aquasecurity/trivy-action@0.28.0 aquasecurity/trivy-action@915b19bbe73b92a6cf82a1bc12b087c9a19a5fe2
docker/login-action@v3 docker/login-action@c94ce9fb468520275223c153574b00df6fe4bcc9
docker/setup-buildx-action@v3 docker/setup-buildx-action@8d2750c68a42422c14e847fe6c8ac0403b4cbd6f
peaceiris/actions-gh-pages@v3 peaceiris/actions-gh-pages@373f7f263a76c20808c831209c920827a82a2847
pypa/gh-action-pypi-publish@release/v1 pypa/gh-action-pypi-publish@ed0c53931b1dc9bd32cbe73a98c7f6766f8a527e
softprops/action-gh-release@v2 softprops/action-gh-release@153bb8e04406b158c6c84fc1615b65b24149a1fe
EOF
)

for workflow in .github/workflows/*.yml; do
  echo "Processing $workflow..."
  while read -r old new; do
    [ -n "${old:-}" ] || continue
    if grep -q "$old" "$workflow"; then
      echo "  Replacing: $old -> $new"
      sed -i.bak "s|$old|$new # ${old#*@}|g" "$workflow"
      rm -f "${workflow}.bak"
    fi
  done <<EOF
$PINS
EOF
done

echo "All actions pinned to SHA."
