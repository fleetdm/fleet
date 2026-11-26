#!/usr/bin/env bash
#
# Runs the Fleet Android agent on an emulator against a local Fleet server and checks that its vitals reach
# Fleet, including the report sent when roaming changes.
#
#   emulator-vitals-test.sh --fleet-url URL --api-token TOKEN [--mysql-db NAME] [--ui] [--avd NAME]
#   emulator-vitals-test.sh --down
#
# --fleet-url   The Fleet server as this machine reaches it, e.g. http://localhost:8080. The emulator reaches
#               the same server through 10.0.2.2.
# --api-token   An admin API token, used to read the enroll secret and the host, and to delete the host.
# --mysql-db    Fleet's MySQL database (default: fleet). Outside AMAPI, Fleet can't create an Android host,
#               so the script inserts one. Dev databases only. Other mysql settings come from MYSQL_ARGS
#               (default: -h127.0.0.1 -P3306 -uroot -ptoor).
# --ui          Boot with a window and stop after enrolling, for testing by hand. Toggle roaming with
#               "roaming on" / "roaming off" below, then clean up with --down.
# --avd         The AVD to use (default: fleet-agent-test). It's created on first use from the Google APIs
#               system image, which allows adb root; Google Play images don't, and roaming can't be
#               simulated without it.
#
#   emulator-vitals-test.sh roaming on|off
#
# Forces the emulator's network to roaming or home. The emulator's own roaming toggle has no effect, because
# Android ignores roaming on the SIM's home network, so this marks that network as roaming in carrier config.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ANDROID_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ANDROID_HOME="${ANDROID_HOME:-$HOME/Library/Android/sdk}"
EMULATOR="$ANDROID_HOME/emulator/emulator"
AVDMANAGER="$ANDROID_HOME/cmdline-tools/latest/bin/avdmanager"
SYSTEM_IMAGE="${SYSTEM_IMAGE:-system-images;android-36.1;google_apis;arm64-v8a}"
STATE_FILE="${TMPDIR:-/tmp}/fleet-agent-emulator-test.state"
MYSQL_ARGS="${MYSQL_ARGS:--h127.0.0.1 -P3306 -uroot -ptoor}"
APP_ID="com.fleetdm.agent"
HOME_PLMN="310260" # the emulator's simulated carrier

AVD="fleet-agent-test"
FLEET_URL=""
API_TOKEN=""
MYSQL_DB="fleet"
UI=false
STARTED_EMULATOR=false

log() { printf '==> %s\n' "$*" >&2; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

set_roaming() {
	adb root >/dev/null
	adb wait-for-device
	if [ "$1" = "on" ]; then
		adb shell cmd phone cc set-value -p gsm_roaming_networks_string_array "$HOME_PLMN" >/dev/null
	else
		adb shell cmd phone cc clear-values >/dev/null
	fi
	log "roaming $1"
}

fleet_api() {
	local method="$1" path="$2"
	curl -fsS -X "$method" -H "Authorization: Bearer $API_TOKEN" "$FLEET_URL$path"
}

down() {
	[ -f "$STATE_FILE" ] || die "nothing to clean up ($STATE_FILE not found)"
	# shellcheck disable=SC1090
	. "$STATE_FILE"
	if [ -n "${HOST_ID:-}" ]; then
		fleet_api DELETE "/api/latest/fleet/hosts/$HOST_ID" >/dev/null && log "deleted host $HOST_ID"
	fi
	if [ "${STARTED_EMULATOR:-false}" = true ]; then
		adb emu kill >/dev/null 2>&1 && log "stopped the emulator"
	fi
	rm -f "$STATE_FILE"
}

ensure_avd() {
	if "$EMULATOR" -list-avds | grep -qx "$AVD"; then
		return
	fi
	log "creating AVD $AVD from $SYSTEM_IMAGE"
	echo no | "$AVDMANAGER" create avd -n "$AVD" -k "$SYSTEM_IMAGE" -d medium_phone >/dev/null
	# avdmanager writes target=android-0 for some images, and the emulator then runs without hardware
	# acceleration ("hvf is not enabled"), so a cold boot takes far too long.
	local platform
	platform="$(printf '%s' "$SYSTEM_IMAGE" | cut -d';' -f2)"
	sed -i.bak "s/^target=.*/target=$platform/" "$HOME/.android/avd/$AVD.ini"
	sed -i.bak 's/^hw.gpu.enabled=no$/hw.gpu.enabled=yes/' "$HOME/.android/avd/$AVD.avd/config.ini"
	rm -f "$HOME/.android/avd/$AVD.ini.bak" "$HOME/.android/avd/$AVD.avd/config.ini.bak"
}

boot_emulator() {
	if adb get-state >/dev/null 2>&1; then
		log "using the running emulator"
		return
	fi
	STARTED_EMULATOR=true
	local window_args=(-no-window)
	$UI && window_args=()
	log "booting $AVD (the first boot is a cold boot and can take several minutes)"
	"$EMULATOR" -avd "$AVD" "${window_args[@]}" -no-audio -no-boot-anim >"${TMPDIR:-/tmp}/fleet-agent-emulator.log" 2>&1 &
	adb wait-for-device
	until [ "$(adb shell getprop sys.boot_completed 2>/dev/null | tr -d '\r')" = "1" ]; do sleep 3; done
	log "booted"
}

create_host() {
	HOST_UUID="$(uuidgen)"
	# shellcheck disable=SC2086
	mysql $MYSQL_ARGS "$MYSQL_DB" -e "INSERT INTO hosts
		(hostname, computer_name, platform, os_version, uuid, hardware_serial, detail_updated_at, label_updated_at, policy_updated_at)
		VALUES ('emulator-$HOST_UUID', 'Android emulator', 'android', 'Android 16', '$HOST_UUID', '$HOST_UUID', NOW(), NOW(), NOW())"
	# shellcheck disable=SC2086
	HOST_ID="$(mysql $MYSQL_ARGS -N -B "$MYSQL_DB" -e "SELECT id FROM hosts WHERE uuid = '$HOST_UUID'")"
	printf 'HOST_ID=%s\nFLEET_URL=%s\nAPI_TOKEN=%s\nSTARTED_EMULATOR=%s\n' "$HOST_ID" "$FLEET_URL" "$API_TOKEN" "$STARTED_EMULATOR" >"$STATE_FILE"
	chmod 600 "$STATE_FILE"
	log "created Android host $HOST_ID ($HOST_UUID)"
}

host_roaming() {
	fleet_api GET "/api/latest/fleet/hosts/$HOST_ID" | jq -r 'if .host | has("roaming") then .host.roaming | tostring else "unreported" end'
}

wait_for_roaming() {
	local want="$1"
	for _ in $(seq 1 60); do
		[ "$(host_roaming)" = "$want" ] && { log "Fleet shows roaming=$want"; return; }
		sleep 2
	done
	die "Fleet still shows roaming=$(host_roaming), expected $want"
}

enroll_and_report() {
	local expect_roaming="$1"
	local device_url
	device_url="$(printf '%s' "$FLEET_URL" | sed -E 's#^(https?://)[^/:]+#\110.0.2.2#')"
	log "running the instrumented test against $device_url"
	local output
	output="$(adb shell am instrument -w \
		-e class "$APP_ID.DeviceVitalsEmulatorTest" \
		-e fleetUrl "$device_url" \
		-e enrollSecret "$ENROLL_SECRET" \
		-e hostUuid "$HOST_UUID" \
		-e expectRoaming "$expect_roaming" \
		"$APP_ID.test/androidx.test.runner.AndroidJUnitRunner")"
	printf '%s\n' "$output" >&2
	printf '%s' "$output" | grep -q "^OK (1 test)" || die "the instrumented test failed"
}

main() {
	case "${1:-}" in
	--down)
		down
		return
		;;
	roaming)
		set_roaming "${2:?on or off}"
		return
		;;
	esac

	while [ $# -gt 0 ]; do
		case "$1" in
		--fleet-url) FLEET_URL="${2%/}"; shift 2 ;;
		--api-token) API_TOKEN="$2"; shift 2 ;;
		--mysql-db) MYSQL_DB="$2"; shift 2 ;;
		--avd) AVD="$2"; shift 2 ;;
		--ui) UI=true; shift ;;
		*) die "unknown argument: $1" ;;
		esac
	done
	[ -n "$FLEET_URL" ] || die "--fleet-url is required"
	[ -n "$API_TOKEN" ] || die "--api-token is required"
	for tool in adb curl jq mysql uuidgen; do
		command -v "$tool" >/dev/null || die "$tool is required"
	done
	[ -f "$STATE_FILE" ] && die "a previous run is still set up; run $0 --down first"

	ENROLL_SECRET="$(fleet_api GET /api/latest/fleet/spec/enroll_secret | jq -r '.spec.secrets[0].secret // empty')"
	[ -n "$ENROLL_SECRET" ] || die "Fleet has no global enroll secret; add one first"

	ensure_avd
	boot_emulator
	set_roaming off
	# Keep cellular data up while roaming, so the agent's network callback sees the change.
	adb shell settings put global data_roaming 1

	log "installing the debug app and tests"
	(cd "$ANDROID_DIR" && ./gradlew -q installDebug installDebugAndroidTest)

	create_host
	enroll_and_report false
	wait_for_roaming false

	log "launching the app"
	adb shell am start -n "$APP_ID/.MainActivity" >/dev/null

	if $UI; then
		log "ready: host $HOST_ID at $FLEET_URL/hosts/$HOST_ID"
		log "toggle roaming with: $0 roaming on|off"
		log "check Fleet with: curl -H 'Authorization: Bearer ...' $FLEET_URL/api/latest/fleet/hosts/$HOST_ID"
		log "agent logs: adb logcat -s fleet-ApiClient fleet-DeviceVitalsWorker"
		log "clean up with: $0 --down"
		return
	fi

	set_roaming on
	wait_for_roaming true
	set_roaming off
	wait_for_roaming false
	log "passed"
	down
}

main "$@"
