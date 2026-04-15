#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Robin Jarry
#
# Update all direct dependencies to the latest version compatible with the
# minimum go version declared in go.mod.

set -eu -o pipefail

go_version=$(sed -En 's/^go ([0-9.]+)$/\1/p' go.mod)
if [ -z "$go_version" ]; then
	echo "error: failed to read go version from go.mod" >&2
	exit 1
fi
echo "go version: $go_version"

module=$(go list -m -f '{{.Path}}')

# version_compatible checks if the go directive from a dependency's go.mod is
# compatible with (i.e. less than or equal to) our minimum go version.
version_compatible() {
	local dep_go="$1"
	if [ -z "$dep_go" ]; then
		# no go directive means compatible with anything
		return 0
	fi
	local min
	min=$(printf '%s\n%s\n' "$go_version" "$dep_go" | sort -V | head -1)
	[ "$min" = "$dep_go" ]
}

# dep_go_version downloads a module at a given version and reads its go.mod go
# directive. Returns empty string if no go directive is found. Returns an error
# if download fails.
dep_go_version() {
	local mod_file
	mod_file=$(go mod download -json "$1" | jq -r .GoMod)
	if [ -z "$mod_file" ]; then
		return 1
	fi
	sed -En 's/^go ([0-9.]+)$/\1/p' "$mod_file"
}

# find_latest_compatible iterates through all versions of a module from newest
# to oldest and returns the first one whose go directive is compatible.
find_latest_compatible() {
	local versions
	# reverse the list: newest first
	versions=$(go list -m -versions -json "$1" | jq -r '.Versions|reverse[]')
	if [ -z "$versions" ]; then
		return
	fi

	for ver in $versions; do
		local dep_go
		dep_go=$(dep_go_version "$dep@$ver") || continue
		if version_compatible "$dep_go"; then
			echo "$ver"
			return
		fi
	done
}

deps=$(go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all | grep -v "^$module$")

for dep in $deps; do
	echo -n "$dep ... "
	best=$(find_latest_compatible "$dep")
	if [ -z "$best" ]; then
		echo "no compatible version found!"
		continue
	fi
	cur=$(go list -m -f '{{.Version}}' "$dep" 2>/dev/null || true)
	if [ "$cur" = "$best" ]; then
		echo "$best (unchanged)"
	else
		echo "$cur -> $best"
		go get "$dep@$best"
	fi
done

# ensure go directive and toolchain are not modified
go mod edit -go="$go_version" -toolchain=none
go mod tidy -go="$go_version" -compat="$go_version"
