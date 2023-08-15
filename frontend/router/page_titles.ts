// Note: Dynamic page titles are constructed for host, software, query, and policy details on their respective *DetailsPage.tsx file
// Note: Order matters for use of array.find() (specific subpaths must be listed before their parent path)
export default [
  { path: "/dashboard", title: "Dashboard | Fleet for osquery" },
  { path: "/hosts/manage", title: "Manage hosts | Fleet for osquery" },
  {
    path: "/controls/mac-os-updates",
    title: "Manage macOS updates | Fleet for osquery",
  },
  {
    path: "/controls/mac-settings",
    title: "Manage macOS settings | Fleet for osquery",
  },
  {
    path: "/controls/mac-setup",
    title: "Manage macOS MDM setup | Fleet for osquery",
  },
  { path: "/software/manage", title: "Manage software | Fleet for osquery" },
  { path: "/queries/manage", title: "Manage queries | Fleet for osquery" },
  { path: "/queries/new", title: "New query | Fleet for osquery" },
  { path: "/policies/manage", title: "Manage policies | Fleet for osquery" },
  { path: "/policies/new", title: "New policy | Fleet for osquery" },
  {
    path: "/settings/organization",
    title: "Manage organization settings | Fleet for osquery",
  },
  {
    path: "/settings/integrations",
    title: "Manage integration settings | Fleet for osquery",
  },
  {
    path: "/settings/users",
    title: "Manage user settings | Fleet for osquery",
  },
  {
    path: "/settings/teams/members",
    title: "Manage team members | Fleet for osquery",
  },
  {
    path: "/settings/teams/options",
    title: "Manage team options | Fleet for osquery",
  },
  {
    path: "/settings/teams",
    title: "Manage team settings | Fleet for osquery",
  },
  {
    path: "/profile",
    title: "Manage my account | Fleet for osquery",
  },
];
