// Helper variables to make writing frequently used roles more concise
const allAdmins = ["global-admin", "team-admin"];
const allMaintainers = ["global-maintainer", "team-maintainer"];

// This is a mapping of the application actions to the roles that have
// permission to perform them.
export const permissions = {
  // host actions
  "host.create": [...allAdmins, ...allMaintainers],

  // label actions
  "label.create": ["global-admin", "global-maintainer"],
};

export type Permission = keyof typeof permissions;
