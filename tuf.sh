SYSTEMS="linux linux-arm64" \
DEB_FLEET_URL=https://host.docker.internal:8080 \
DEB_TUF_URL=http://host.docker.internal:8081 \
GENERATE_DEB=1 \
GENERATE_DEB_ARM64=1 \
ENROLL_SECRET=5l/k8l8c5Cogs//2PJFI3yxVuYpOCCZR \
USE_FLEET_SERVER_CERTIFICATE=1 \
FLEET_DESKTOP=1 \
DEBUG=1 \
./tools/tuf/test/main.sh
