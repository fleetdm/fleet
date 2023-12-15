// Note: Dynamic page titles are constructed for host, software, query, and policy details on their respective *DetailsPage.tsx file

import { DOCUMENT_TITLE_SUFFIX } from "utilities/constants";
import PATHS from "router/paths";

// Note: Order matters for use of array.find() (specific subpaths must be listed before their parent path)
export default [
  { path: "/dashboard", title: `Dashboard | ${DOCUMENT_TITLE_SUFFIX}` },
  { path: "/hosts/manage", title: `Manage hosts | ${DOCUMENT_TITLE_SUFFIX}` },
  {
    path: "/controls/os-updates",
    title: `Manage OS updates | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: "/controls/os-settings",
    title: `Manage OS settings | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: "/controls/setup-experience",
    title: `Manage setup experience | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: PATHS.SOFTWARE_TITLES,
    title: `Software titles | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: PATHS.SOFTWARE_VERSIONS,
    title: `Software versions | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: "/queries/manage",
    title: `Manage queries | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  { path: "/queries/new", title: `New query | ${DOCUMENT_TITLE_SUFFIX}` },
  {
    path: "/policies/manage",
    title: `Manage policies | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  { path: "/policies/new", title: `New policy | ${DOCUMENT_TITLE_SUFFIX}` },
  {
    path: "/settings/organization",
    title: `Manage organization settings | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: "/settings/integrations",
    title: `Manage integration settings | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: "/settings/users",
    title: `Manage user settings | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: "/settings/teams/members",
    title: `Manage team members | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: "/settings/teams/options",
    title: `Manage team options | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: "/settings/teams",
    title: `Manage team settings | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: "/profile",
    title: `Manage my account | ${DOCUMENT_TITLE_SUFFIX}`,
  },
];
