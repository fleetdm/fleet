import PATHS from "router/paths";
import URL_PREFIX from "router/url_prefix";
import { IUser } from "interfaces/user";

export interface INavItem {
  icon: string;
  name: string;
  iconName: string;
  location: {
    regex: RegExp;
    pathname: string;
  };
  withContext?: boolean;
}

export default (user: IUser | null): INavItem[] => {
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

  return [...logo];
};
