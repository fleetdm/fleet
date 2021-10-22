import React from "react";
import { useDispatch, useSelector } from "react-redux";
import { push } from "react-router-redux";

import { IConfig } from "interfaces/config";
import permissionUtils from "utilities/permissions";
import paths from "router/paths";

interface IPremiumTierRoutes {
  children: JSX.Element;
}

interface IRootState {
  app: {
    config: IConfig;
  };
}

const { FLEET_403 } = paths;

const PremiumTierRoutes = ({
  children,
}: IPremiumTierRoutes): JSX.Element | null => {
  const dispatch = useDispatch();
  const config = useSelector((state: IRootState) => state.app.config);

  // config is an empty object here. The API result has not come back
  // so render nothing.
  if (Object.keys(config).length === 0) {
    return null;
  }

  if (!permissionUtils.isPremiumTier(config)) {
    dispatch(push(FLEET_403));
    return null;
  }
  return <>{children}</>;
};

export default PremiumTierRoutes;
