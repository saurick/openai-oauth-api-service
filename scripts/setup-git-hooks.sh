#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

chmod +x .githooks/pre-commit .githooks/pre-push .githooks/commit-msg
chmod +x scripts/git-hooks/pre-commit.sh scripts/git-hooks/pre-push.sh scripts/git-hooks/commit-msg.sh
chmod +x scripts/setup-git-hooks.sh scripts/bootstrap.sh scripts/doctor.sh
chmod +x scripts/qa/full.sh scripts/qa/fast.sh scripts/qa/strict.sh scripts/qa/db-guard.sh scripts/qa/secrets.sh scripts/qa/shellcheck.sh
chmod +x scripts/qa/go-vet.sh scripts/qa/golangci-lint.sh scripts/qa/yamllint.sh
chmod +x scripts/qa/shfmt.sh scripts/qa/govulncheck.sh

git config core.hooksPath .githooks

echo "Git hooks 已启用：core.hooksPath=.githooks"
echo "可通过 git config --get core.hooksPath 验证"
