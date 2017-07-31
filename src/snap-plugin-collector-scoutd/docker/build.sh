#!/bin/bash

set -e
set -u
set -o pipefail

os=${1:-"xenial"}
org="solarwinds"

cmd="docker build -t ${org}/snap_scout:${os} \
  --build-arg BUILD_DATE=$(date +%Y-%m-%d)"

${cmd} -f "${os}/Dockerfile" .
