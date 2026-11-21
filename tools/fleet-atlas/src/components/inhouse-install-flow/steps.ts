import type { SortStep } from "@/components/flow-sorts/SortQuiz";

// In-house (private IPA) install lifecycle on Apple. Unlike VPP, Fleet owns
// the bytes and serves them directly. The one-stop-removed Enterprise
// Distribution cert + manifest-with-hash pattern is the thing most people
// forget. If the hash doesn't match the IPA, the install silently fails.

export const INHOUSE_INSTALL_STEPS: SortStep[] = [
  {
    key: "admin-uploads",
    actor: "Admin",
    text: "Admin uploads the signed IPA + manifest; Fleet validates the signing, stores bytes in the file store, writes metadata to in_house_apps.",
  },
  {
    key: "admin-assigns",
    actor: "Admin",
    text: "Admin assigns the in-house app to a fleet; Fleet writes a software_titles row + fleet association.",
  },
  {
    key: "admin-installs",
    actor: "Admin",
    text: "Admin clicks Install on a host; Fleet writes a host_in_house_software_installs row with status=pending + an in_house_app_upcoming_activities entry.",
  },
  {
    key: "apns-wake",
    actor: "Fleet",
    text: "apple_mdm_apns_pusher sends an APNs push to wake the device.",
  },
  {
    key: "mdm-command",
    actor: "Fleet",
    text: "Device checks in; Fleet responds with an InstallEnterpriseApplication command carrying a Fleet-served manifest URL + expected IPA hash.",
  },
  {
    key: "device-downloads",
    actor: "Device",
    text: "The OS's MDM client fetches the manifest and the IPA bytes from Fleet's endpoint.",
  },
  {
    key: "device-installs",
    actor: "Device",
    text: "Device installs the IPA; the OS validates the signature against the enterprise distribution cert before completing.",
  },
  {
    key: "ack-status",
    actor: "Device",
    text: "Device reports install status back via MDM ACK; Fleet updates host_in_house_software_installs to installed.",
  },
];
