#!/usr/bin/env bash

set -eo pipefail

usage() {
  base="$(basename "$0")"
  cat <<EOUSAGE
Usage: ${base} [args]
  -C,--ci               : Replicate full Circle CI run
  -D,--deps             : Build dependencies
  -B,--build            : Build a release
  -T,--test             : Run 'make test'
  make [option]         : Run any kolide Makefile command
EOUSAGE
}

if [ $# -eq 0 ]; then
  usage
fi

# Flag parsing
while [[ $# -gt 0 ]]; do
  opt="$1"
  case "${opt}" in
    -C|--ci)
      ci=1
      shift
      ;;
    -B|--build)
      build=1
      shift
      ;;
    -D|--deps)
      deps=1
      shift
      ;;
    -T|--tests)
      tests=1
      shift
      ;;
    make)
      make $2
      shift
      ;;
    *)
      echo "Error: Unknown option: ${opt}"
      usage
      exit 1
      ;;
  esac
done

function ci_run {
    echo 'running full circle ci test suite'
    echo "make deps"
    make deps

    echo "make generate"
    make generate

    echo "make test"
    make test

    echo "make build"
    make build
}

deps=${deps:-0}
if [ ${deps} -eq 1 ]; then
    make deps
    make generate
    GOGC=off go install
  exit 0
fi

build=${build:-0}
if [ ${build} -eq 1 ]; then
    make test
    make build
  exit 0
fi

tests=${tests:-0}
if [ ${tests} -eq 1 ]; then
    make test
  exit 0
fi

ci=${ci:-0}
if [ ${ci} -eq 1 ]; then
    ci_run
  exit 0
fi
