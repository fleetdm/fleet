import type { SortStep } from "@/components/flow-sorts/SortQuiz";

// Managed Google Play lifecycle: Fleet never talks directly to Android devices.
// Fleet writes policy, calls AMAPI, Google delivers, Google Play installs,
// Google reports completion back via Pub/Sub. The one-step-removed shape is
// the Android MDM thing that trips up anyone used to Apple's direct MDM.

export const MANAGED_GOOGLE_PLAY_STEPS: SortStep[] = [
  {
    key: "one-time-setup",
    actor: "Admin",
    text: "One-time: admin links Fleet to their Managed Google Play enterprise; Fleet receives credentials to call AMAPI.",
  },
  {
    key: "catalog-browse",
    actor: "Admin",
    text: "Admin picks an app from the Managed Google Play catalog in Fleet's UI; Fleet queries AMAPI for the app's metadata.",
  },
  {
    key: "assign-to-fleet",
    actor: "Fleet",
    text: "Admin assigns the app to a fleet; Fleet writes it into the policy document that governs every host in that fleet.",
  },
  {
    key: "policies-patch",
    actor: "Fleet",
    text: "Fleet calls Google's enterprises.policies.patch to push the updated policy to the enterprise.",
  },
  {
    key: "google-pushes",
    actor: "Google",
    text: "Google's push service delivers the policy update to the Google Device Policy app on the device.",
  },
  {
    key: "play-installs",
    actor: "Device",
    text: "Google Play on the device sees the policy, pulls the app from the Play Store, and installs it in the work container.",
  },
  {
    key: "pubsub-callback",
    actor: "Google",
    text: "Google calls Fleet's /api/v1/fleet/android_enterprise/pubsub endpoint via Pub/Sub when the install completes.",
  },
  {
    key: "fleet-updates-status",
    actor: "Fleet",
    text: "Fleet updates host_mdm_android to reflect the new install.",
  },
];
