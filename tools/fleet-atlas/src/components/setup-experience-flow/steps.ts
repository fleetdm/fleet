import type { SortStep } from "@/components/flow-sorts/SortQuiz";

// Setup experience: Fleet pre-installs apps during Mac setup assistant so new
// devices arrive ready. The "awaiting_configuration" row + DDM release are
// the Fleet-specific pieces.

export const SETUP_EXPERIENCE_STEPS: SortStep[] = [
  {
    key: "admin-configures",
    actor: "Admin",
    text: "Admin configures setup experience for the fleet: bootstrap package, required titles marked 'install at setup,' post-setup script.",
  },
  {
    key: "user-unboxes",
    actor: "External",
    text: "End user unboxes a DEP-assigned Mac, powers on; MDM enrollment completes.",
  },
  {
    key: "enqueue-awaiting",
    actor: "Fleet",
    text: "At TokenUpdate checkin, EnqueueSetupExperienceItems writes a host_mdm_apple_awaiting_configuration row and stages the required installs + scripts.",
  },
  {
    key: "bootstrap-install",
    actor: "Fleet",
    text: "Fleet pushes the bootstrap package via InstallEnterpriseApplication; fleetd gets installed on the device.",
  },
  {
    key: "orbit-config",
    actor: "Device",
    text: "fleetd starts; its first /orbit/config call sees RunSetupExperience=true and begins pulling the queued installs.",
  },
  {
    key: "installs-run",
    actor: "Device",
    text: "fleetd downloads bytes and runs pre/install/post scripts for each setup item; each result flips a setup_experience_status_results row.",
  },
  {
    key: "release-device",
    actor: "Fleet",
    text: "When every required item reports success, Fleet releases the device via DDM (SetHostAwaitingConfiguration false).",
  },
  {
    key: "login-screen",
    actor: "Device",
    text: "Mac finishes setup assistant and shows the login screen; required software is already present.",
  },
];
