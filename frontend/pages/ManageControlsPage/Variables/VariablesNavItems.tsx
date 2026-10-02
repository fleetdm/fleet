import { InjectedRouter } from "react-router";

import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";
import PATHS from "router/paths";

import CustomHostVitalsTab from "./cards/CustomHostVitalsTab";
import GlobalVariables from "./cards/GlobalVariables";

export interface IVariablesCardProps {
  router: InjectedRouter;
  location: {
    pathname: string;
    query: {
      add_variable?: string;
      query?: string;
      page?: string;
      order_key?: string;
      order_direction?: string;
    };
  };
}

const getVariablesNavItems = (): ISideNavItem<IVariablesCardProps>[] => {
  return [
    {
      title: "Global variables",
      urlSection: "global-variables",
      path: PATHS.CONTROLS_VARIABLES_GLOBAL_VARIABLES,
      Card: GlobalVariables,
    },
    {
      title: "Custom host vitals",
      urlSection: "custom-host-vitals",
      path: PATHS.CONTROLS_VARIABLES_CUSTOM_HOST_VITALS,
      Card: CustomHostVitalsTab,
    },
  ];
};

export default getVariablesNavItems;
