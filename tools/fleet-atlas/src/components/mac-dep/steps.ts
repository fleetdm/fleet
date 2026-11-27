import type { SortStep } from "@/components/flow-sorts/SortQuiz";

// Zero-touch Mac enrollment via Apple's Device Enrollment Program (DEP).
// Written in the deterministic order the flow actually happens. Quiz shuffles
// them at render time.

export const MAC_DEP_STEPS: SortStep[] = [
  {
    key: "abm-assign",
    actor: "Admin",
    text: "IT buys the Mac through Apple Business Manager and assigns it to Fleet's MDM server in ABM.",
  },
  {
    key: "unbox",
    actor: "External",
    text: "End user unboxes the Mac; on first boot it contacts Apple's activation servers.",
  },
  {
    key: "apple-handoff",
    actor: "Apple",
    text: "Apple sees the serial is assigned to Fleet and tells the Mac: 'your MDM is at fleet.example.com.'",
  },
  {
    key: "mac-hits-enroll",
    actor: "Device",
    text: "Mac hits Fleet's enrollment endpoint (/mdm/apple/mdm and related DEP paths).",
  },
  {
    key: "fleet-returns-profile",
    actor: "Fleet",
    text: "Fleet verifies the DEP challenge and responds with an enrollment profile (XML plist).",
  },
  {
    key: "mac-installs-profile",
    actor: "Device",
    text: "Mac installs the profile; the MDM channel is now open.",
  },
  {
    key: "fleet-writes-rows",
    actor: "Fleet",
    text: "Fleet writes a host_dep_assignments row and nano_enrollments row to persist the enrollment.",
  },
  {
    key: "setup-assistant",
    actor: "Device",
    text: "Setup assistant runs; bootstrap package installs; initial config profiles push.",
  },
];
