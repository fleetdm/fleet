import PATHS from "router/paths";
import URL_PREFIX from "router/url_prefix";
import permissionUtils from "utilities/permissions";

export default (currentUser) => {
  const userNavItems = [
    {
      icon: "logo",
      name: "Home",
      iconName: "logo",
      location: {
        regex: new RegExp(`^${URL_PREFIX}/dashboard`),
        pathname: PATHS.HOME,
      },
    },
    {
      icon: "hosts",
      name: "Hosts",
      iconName: "hosts",
      location: {
        regex: new RegExp(`^${URL_PREFIX}/hosts/`),
        pathname: PATHS.MANAGE_HOSTS,
      },
    },
    {
      icon: "query",
      name: "Queries",
      iconName: "queries",
      location: {
        regex: new RegExp(`^${URL_PREFIX}/queries/`),
        pathname: PATHS.MANAGE_QUERIES,
      },
    },
  ];

  const policiesTab = [
    {
      icon: "policies",
      name: "Policies",
      iconName: "policies",
      location: {
        regex: new RegExp(`^${URL_PREFIX}/(policies)/`),
        pathname: PATHS.MANAGE_POLICIES,
      },
    },
  ];

  const teamMaintainerNavItems = [
    {
      icon: "packs",
      name: "Schedule",
      iconName: "packs",
      location: {
        regex: new RegExp(`^${URL_PREFIX}/(schedule|packs)/`),
        pathname: PATHS.MANAGE_SCHEDULE,
      },
    },
  ];

  if (
    permissionUtils.isAnyTeamAdmin(currentUser) ||
    permissionUtils.isGlobalAdmin(currentUser)
  ) {
    const adminNavItems = [
      {
        icon: "settings",
        name: "Settings",
        iconName: "settings",
        location: {
          regex: new RegExp(`^${URL_PREFIX}/settings/`),
          pathname:
            currentUser.global_role === "admin"
              ? PATHS.ADMIN_SETTINGS
              : `${PATHS.ADMIN_TEAMS}/${currentUser.teams[0].id}/members`,
        },
      },
    ];
    return [
      ...userNavItems,
      ...teamMaintainerNavItems,
      ...policiesTab,
      ...adminNavItems,
    ];
  }

  if (
    permissionUtils.isGlobalMaintainer(currentUser) ||
    permissionUtils.isAnyTeamMaintainer(currentUser)
  ) {
    return [...userNavItems, ...teamMaintainerNavItems, ...policiesTab];
  }

  return [...userNavItems, ...policiesTab];
};
