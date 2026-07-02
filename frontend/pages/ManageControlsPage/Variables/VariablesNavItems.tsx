import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";

import GlobalVariables from "./cards/GlobalVariables";
import CustomHostVitalsTab from "./cards/CustomHostVitalsTab";

export interface IVariablesCardProps {
  router: InjectedRouter;
  location: {
    pathname: string;
    query: { add_variable?: string };
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
