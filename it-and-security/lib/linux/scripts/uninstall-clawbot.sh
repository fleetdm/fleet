#!/bin/bash

# Uninstall Clawbot from Linux

if command -v apt-get &> /dev/null; then
  apt-get remove -y clawbot 2>/dev/null || true
  apt-get purge -y clawbot 2>/dev/null || true
fi

if command -v dnf &> /dev/null; then
  dnf remove -y clawbot 2>/dev/null || true
elif command -v yum &> /dev/null; then
  yum remove -y clawbot 2>/dev/null || true
fi

exit 0
