// Note: Dynamic page titles are constructed for host, software, query, and policy details on their respective *DetailsPage.tsx file

import { DOCUMENT_TITLE_SUFFIX } from "utilities/constants";
import PATHS from "router/paths";

// Note: Order matters for use of array.find() (specific subpaths must be listed before their parent path)
export default [
  { path: PATHS.DASHBOARD, title: `Dashboard | ${DOCUMENT_TITLE_SUFFIX}` },
  { path: "/hosts/manage", title: `Manage hosts | ${DOCUMENT_TITLE_SUFFIX}` },
  {
    path: PATHS.CONTROLS_OS_UPDATES,
    title: `Manage OS updates | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: PATHS.CONTROLS_OS_SETTINGS,
    title: `Manage OS settings | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: PATHS.CONTROLS_SETUP_EXPERIENCE,
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
    path: PATHS.MANAGE_QUERIES,
    title: `Manage queries | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  { path: PATHS.NEW_QUERY(), title: `New query | ${DOCUMENT_TITLE_SUFFIX}` },
  {
    path: PATHS.MANAGE_POLICIES,
    title: `Manage policies | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  { path: PATHS.NEW_POLICY, title: `New policy | ${DOCUMENT_TITLE_SUFFIX}` },
  {
    path: PATHS.ADMIN_ORGANIZATION,
    title: `Manage organization settings | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: PATHS.ADMIN_INTEGRATIONS,
    title: `Manage integration settings | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: PATHS.ADMIN_USERS,
    title: `Manage user settings | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: PATHS.TEAM_DETAILS_MEMBERS(),
    title: `Manage team members | ${DOCUMENT_TITLE_SUFFIX}`,
  },
  {
    path: PATHS.TEAM_DETAILS_OPTIONS(),
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
