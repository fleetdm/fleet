import PATHS from "router/paths";
import URL_PREFIX from "router/url_prefix";
import permissionUtils from "utilities/permissions";

export default (currentUser) => {
  const adminNavItems = [
    {
      icon: "settings",
      name: "Settings",
      iconName: "settings",
      location: {
        regex: new RegExp(`^${URL_PREFIX}/settings/`),
        pathname: PATHS.ADMIN_SETTINGS,
      },
    },
  ];

  const userNavItems = [
    {
      icon: "logo",
      name: "Home",
      iconName: "logo",
      location: {
        regex: new RegExp(`^${URL_PREFIX}/home/dashboard`),
        pathname: PATHS.HOMEPAGE,
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

  const globalMaintainerNavItems = [
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

  if (permissionUtils.isGlobalAdmin(currentUser)) {
    return [...userNavItems, ...globalMaintainerNavItems, ...adminNavItems];
  }

  if (permissionUtils.isGlobalMaintainer(currentUser)) {
    return [...userNavItems, ...globalMaintainerNavItems];
  }

  return userNavItems;
};
