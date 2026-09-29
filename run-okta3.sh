#!/bin/bash
{
echo "== amd64 download path (checksum + unzip only)"
multipass exec okta-test -- bash -c 'd=$(mktemp -d); cd $d && curl -fsSLO https://github.com/micromdm/scep/releases/download/v2.3.0/scepclient-linux-amd64-v2.3.0.zip && echo "47b20e1b44b5789d4dd8572b17b22fe7d7d5fa6c1fe1b61f24fe26a3a1826a30  scepclient-linux-amd64-v2.3.0.zip" | sha256sum -c - && (command -v unzip >/dev/null || sudo apt-get install -y -q unzip >/dev/null) && unzip -q scepclient-linux-amd64-v2.3.0.zip && file scepclient-linux-amd64; rm -rf $d'
echo "== full run (arm64 build path)"
multipass exec okta-test -- sudo bash -c 'rm -rf /tmp/okta-scep.* /usr/local/lib/okta-scep /usr/local/bin/scepclient'
multipass transfer ~/Downloads/okta-scep-enroll.sh okta-test: \
  && multipass exec okta-test -- chmod +x okta-scep-enroll.sh \
  && time multipass exec okta-test -- sudo ./okta-scep-enroll.sh </dev/null
echo "exit=$?"
echo "== second run (should reuse the installed scepclient)"
multipass exec okta-test -- sudo ./okta-scep-enroll.sh </dev/null 2>&1 | grep -iE 'scepclient|Installed|ERROR'
echo "exit2=${PIPESTATUS[0]}"
multipass exec okta-test -- ls -l /usr/local/lib/okta-scep
} > run-okta3.out 2>&1
echo "wrote run-okta3.out"
