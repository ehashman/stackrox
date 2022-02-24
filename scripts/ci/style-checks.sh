#!/usr/bin/env bash

set -euo pipefail

# Run style checks

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"

source "$SCRIPTS_ROOT/scripts/lib.sh"

style_checks() {
    if is_GITHUB_ACTIONS; then
        require_environment "ORG_TOKEN_FOR_GITHUB"
        git config --global "url.https://${ORG_TOKEN_FOR_GITHUB}:x-oauth-basic@github.com/.insteadOf" https://github.com/
    fi

    set -x

    env | sort
    go env GOCACHE
    go env GOMODCACHE

    # make deps
    # make golangci-lint
    make style || true
    find / -name mod -print
    find / -name .cache -print -exec ls -l '{}' \;
    find / -name caches -print
    find / -name node_modules -print
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    style_checks "$*"
fi
