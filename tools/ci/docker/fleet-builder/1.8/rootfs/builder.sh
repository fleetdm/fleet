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
  make [option]         : Run any fleet Makefile command
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
    # Copy SSH key
    mkdir -p /root/.ssh -m 0700
    echo 'github.com ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A7hRGmdnm9tUDbO9IDSwBK6TbQa+PXYPCPy6rbTrTtw7PHkccKrpp0yVhp5HdEIcKr6pLlVDBfOLX9QUsyCOV0wzfjIJNlGEYsdlLJizHhbn2mUjvSAHQqZETYP81eFzLQNnPHt4EVVUh7VfDESU84KezmD5QlWpXLmvU31/yMf+Se8xhHTvKSCZIFImWwoG6mbUoWf9nzpIoaSjB+weqqUUmpaaasXVal72J+UX2B+2RPW3RcT0eOzQgqlJL3RKrTJvdsjE3JEAvGq3lGHSZXy28G3skua2SmVi/w4yCE6gbODqnTWlg7+wC604ydGXA8VJiS5ap43JXiUFFAaQ==
    bitbucket.org ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAubiN81eDcafrgMeLzaFPsw2kNvEcqTKl/VqLat/MaB33pZy0y3rJZtnqwR2qOOvbwKZYKiEO1O6VqNEBxKvJJelCq0dTXWT5pbO2gDXC6h6QDXCaHo6pOHGPUy+YBaGQRGuSusMEASYiWunYN0vCAI8QaXnWMXNMdFP3jHAJH0eDsoiGnLPBlBp4TNm6rYI74nMzgz3B9IikW4WVK+dc8KZJZWYjAuORU3jc1c/NPskD2ASinf8v3xnfXeukU0sJ5N6m5E8VLjObPEO+mN2t/FZTMZLiFqPWc/ALSqnMnnhwrNi2rbfg/rd/IpL8Le3pSBne8+seeFVBoGqzHM9yXw==
    ' >> ~/.ssh/known_hosts
    cp /tmp/id_rsa /root/.ssh/id_rsa
    chmod 0600 /root/.ssh/id_rsa

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
