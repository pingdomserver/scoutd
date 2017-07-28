#!/bin/bash

set -e
set -u
set -o pipefail

os=$1
org=${2:-"solarwinds"}

cmd="docker build -t ${org}/snap_scout:${os} \
  --build-arg BUILD_DATE=$(date +%Y-%m-%d)"

${cmd} -f "${os}/Dockerfile" .
