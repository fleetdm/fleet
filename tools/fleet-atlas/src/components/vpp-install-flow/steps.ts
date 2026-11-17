import type { SortStep } from "@/components/flow-sorts/SortQuiz";

// VPP install flow: Fleet never touches the app bytes. Fleet tells Apple to
// assign a volume license, Apple pushes the install, device acks back through
// MDM. The license-assign step is the one everyone forgets.

export const VPP_INSTALL_STEPS: SortStep[] = [
  {
    key: "admin-click-install",
    actor: "Admin",
    text: "Admin opens Host details → Software, picks the VPP app, clicks Install.",
  },
  {
    key: "service-recognizes-vpp",
    actor: "Fleet",
    text: "Service recognizes the title is VPP-backed; writes upcoming_activities + vpp_app_upcoming_activities with status=pending.",
  },
  {
    key: "worker-calls-vpp-api",
    actor: "Fleet",
    text: "The apple_software worker job picks up the row and calls Apple's VPP API: 'assign this app license to this device identifier.'",
  },
  {
    key: "apple-pushes-install",
    actor: "Apple",
    text: "Apple accepts the assignment and sends an InstallApplication MDM command to the device via APNs.",
  },
  {
    key: "device-downloads",
    actor: "Device",
    text: "Device wakes, downloads the app from Apple's App Store CDN, installs it, ACKs via MDM.",
  },
  {
    key: "fleet-updates-status",
    actor: "Fleet",
    text: "On MDM ACK, Fleet updates host_vpp_software_installs to installed.",
  },
  {
    key: "detail-queries-refresh",
    actor: "Device",
    text: "Next osquery detail-queries refresh host_software; Fleet's software inventory now reflects the install.",
  },
];
