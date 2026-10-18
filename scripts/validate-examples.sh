#!/usr/bin/env bash
# Validate every scenario under examples/scenarios with the locally built
# provider. No AxonOps server or credentials are needed.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT

(cd "${root}" && go build -o "${work}/terraform-provider-axonops" .)

cat > "${work}/terraformrc" <<EOF
provider_installation {
  dev_overrides {
    "axonops/axonops" = "${work}"
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE="${work}/terraformrc"

shopt -s nullglob
dirs=("${root}"/examples/scenarios/*/)
if [[ ${#dirs[@]} -eq 0 ]]; then
  echo "no scenarios found under examples/scenarios" >&2
  exit 1
fi

status=0
for dir in "${dirs[@]}"; do
  name="$(basename "${dir}")"
  echo "validating ${name}"
  # Validate a copy so no .terraform state or lock file lands in the repo.
  cp -R "${dir%/}" "${work}/${name}"
  if ! terraform -chdir="${work}/${name}" validate -no-color; then
    status=1
  fi
done
exit "${status}"
