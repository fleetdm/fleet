// Helper variables to make writing frequently used roles more concise
const allAdmins = ["global-admin", "team-admin"];
const allMaintainers = ["global-maintainer", "team-maintainer"];

// This is a mapping of the application actions to the roles that have
// permission to perform them.

export const freePermissions = {
  // host actions
  "host.create": [...allAdmins, ...allMaintainers],
  "host.delete": ["global-admin", "global-maintainer", "team-admin"],

  // label actions
  "label.create": ["global-admin", "global-maintainer"],
};

export const premiumPermissions = {
  ...freePermissions,

  "host.transfer": ["global-maintainer"],
};

// TODO: Remove this once we have a way to check if a user is on a free or premium tier

export type FreePermissions = keyof typeof freePermissions;
export type PremiumPermissions = keyof typeof premiumPermissions;
