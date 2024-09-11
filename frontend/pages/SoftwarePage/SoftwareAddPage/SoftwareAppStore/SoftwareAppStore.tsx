import React from "react";
import { InjectedRouter } from "react-router";
import { Location } from "history";

import { ISoftwareAddPageQueryParams } from "../SoftwareAddPage";

const baseClass = "software-app-store";

interface ISoftwareAppStoreProps {
  currentTeamId: number;
  router: InjectedRouter;
  location: Location<ISoftwareAddPageQueryParams>;
}

const SoftwareAppStore = ({
  currentTeamId,
  router,
  location,
}: ISoftwareAppStoreProps) => {
  return <div className={baseClass}>Software App store page</div>;
};

export default SoftwareAppStore;
