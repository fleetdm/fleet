import PATHS from "router/paths";
import URL_PREFIX from "router/url_prefix";
import { IUser } from "interfaces/user";
import { isGlobalAdmin } from "utilities/permissions/permissions";

export interface INavItem {
  icon: string;
  name: string;
  iconName: string;
  location: {
    regex: any;
    pathname: string;
  };
}

export default (
  user: IUser | null,
  isGlobalAdmin: boolean = false,
  isAnyTeamAdmin: boolean = false,
  isAnyTeamMaintainer: boolean = false,
  isGlobalMaintainer: boolean = false,
  isNoAccess: boolean = false
) => {
  if (!user) {
    return [];
  }

  const logo = [
    {
      icon: "logo",
      name: "Home",
      iconName: "logo",
      location: {
        regex: new RegExp(`^${URL_PREFIX}/dashboard`),
        pathname: PATHS.HOME,
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
      icon: "software",
      name: "Software",
      iconName: "software",
      location: {
        regex: new RegExp(`^${URL_PREFIX}/software/`),
        pathname: PATHS.MANAGE_SOFTWARE,
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

  const maintainerOrAdminNavItems = [
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
    isGlobalMaintainer ||
    isAnyTeamMaintainer ||
    isGlobalAdmin ||
    isAnyTeamAdmin
  ) {
    return [
      ...logo,
      ...userNavItems,
      ...maintainerOrAdminNavItems,
      ...policiesTab,
    ];
  }

  if (isNoAccess) {
    return [...logo];
  }
  return [...logo, ...userNavItems, ...policiesTab];
};
