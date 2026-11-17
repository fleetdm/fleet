import type { SortStep } from "@/components/flow-sorts/SortQuiz";

// Custom installer lifecycle (.pkg / .msi / .deb / .exe). Fleet owns the
// bytes; fleetd runs the three-script pattern on the device. This is the
// pull-based path. No MDM involved, everything rides on the osquery poll
// cycle.

export const CUSTOM_INSTALL_STEPS: SortStep[] = [
  {
    key: "admin-uploads",
    actor: "Admin",
    text: "Admin uploads the installer (e.g., chrome.pkg) via the UI; Fleet stores bytes in the file store and metadata in software_installers.",
  },
  {
    key: "admin-clicks-install",
    actor: "Admin",
    text: "Admin clicks Install on a host; Fleet writes a host_software_installs row with status=pending + an upcoming_activities entry.",
  },
  {
    key: "fleetd-config-poll",
    actor: "Device",
    text: "On its next POST /api/osquery/config poll (~10s), fleetd sees the pending install work in the config response.",
  },
  {
    key: "pre-install-query",
    actor: "Device",
    text: "fleetd runs the pre_install_query locally; if it returns zero rows, the install is skipped and status flips to failed.",
  },
  {
    key: "download-bytes",
    actor: "Device",
    text: "fleetd downloads the installer bytes from Fleet's endpoint.",
  },
  {
    key: "run-scripts",
    actor: "Device",
    text: "fleetd runs the install_script, then the post_install_script.",
  },
  {
    key: "report-result",
    actor: "Device",
    text: "fleetd POSTs to /api/fleet/orbit/software_install/result with exit codes + stdout/stderr tails.",
  },
  {
    key: "fleet-updates-status",
    actor: "Fleet",
    text: "Fleet updates host_software_installs to installed or failed based on the reported result.",
  },
  {
    key: "detail-queries-refresh",
    actor: "Device",
    text: "Next osquery detail-queries run picks up the new program in host_software inventory.",
  },
];
