#!/bin/bash

# Installs a udev rule that forces USB removable block devices (USB sticks,
# external HDD/SSD presented as removable, SD cards on USB readers, eMMC) to
# read-only at the kernel level. Auto-mounters (udisks2/GNOME, KDE, etc.) will
# then mount these devices read-only because the underlying block device has
# the read-only flag set.

set -e

RULE_PATH="/etc/udev/rules.d/99-fleet-readonly-removable-storage.rules"

if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root." >&2
    exit 1
fi

cat > "$RULE_PATH" <<'EOF'
# Managed by Fleet. Do not edit by hand.
# Forces USB removable storage to read-only by setting the block device RO flag.
ACTION=="add|change", SUBSYSTEMS=="usb", KERNEL=="sd[a-z]",        ATTR{removable}=="1", RUN+="/sbin/blockdev --setro /dev/%k"
ACTION=="add|change", SUBSYSTEMS=="usb", KERNEL=="sd[a-z][0-9]*",                        RUN+="/sbin/blockdev --setro /dev/%k"
ACTION=="add|change", SUBSYSTEMS=="usb", KERNEL=="mmcblk[0-9]*",                         RUN+="/sbin/blockdev --setro /dev/%k"
ACTION=="add|change", SUBSYSTEMS=="usb", KERNEL=="mmcblk[0-9]*p[0-9]*",                  RUN+="/sbin/blockdev --setro /dev/%k"
EOF

chmod 644 "$RULE_PATH"

udevadm control --reload-rules
udevadm trigger --action=change --subsystem-match=block || true

echo "Installed Fleet read-only removable storage udev rule at $RULE_PATH"
