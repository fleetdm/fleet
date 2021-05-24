import PATHS from "router/paths";
import URL_PREFIX from "router/url_prefix";

export default (global_role) => {
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
      name: "Packs",
      iconName: "packs",
      location: {
        regex: new RegExp(`^${URL_PREFIX}/packs/`),
        pathname: PATHS.MANAGE_PACKS,
      },
    },
  ];

  if (global_role === "admin") {
    return [...userNavItems, ...globalMaintainerNavItems, ...adminNavItems];
  }

  if (global_role === "maintainer") {
    return [...userNavItems, ...globalMaintainerNavItems];
  }

  return userNavItems;
};
