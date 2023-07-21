// This is a mapping of the application actions to the roles that have
// permission to perform them.
export const permissions = {
  // host actions
  "host.create": [
    "global-admin",
    "global-maintainer",
    "team-admin",
    "team-maintainer",
  ],

  // label actions
  "label.create": ["global-admin", "global-maintainer"],
};

export type Permission = keyof typeof permissions;
