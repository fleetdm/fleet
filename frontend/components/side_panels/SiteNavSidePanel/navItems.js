import PATHS from "router/paths";
import URL_PREFIX from "router/url_prefix";

export default (admin) => {
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

  if (admin) {
    return [...userNavItems, ...adminNavItems];
  }

  return userNavItems;
};
