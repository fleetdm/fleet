import React from "react";
import { useDispatch, useSelector } from "react-redux";
import { push } from "react-router-redux";

import { IConfig } from "interfaces/config";
import permissionUtils from "utilities/permissions";
import paths from "router/paths";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

interface IBasicTierRoutes {
  children: JSX.Element;
}

interface IRootState {
  app: {
    config: IConfig;
  };
}

const { HOME } = paths;

const BasicTierRoutes = (props: IBasicTierRoutes) => {
  const { children } = props;

  const dispatch = useDispatch();
  const config = useSelector((state: IRootState) => state.app.config);

  if (!config) {
    return null;
  }

  if (!permissionUtils.isBasicTier(config)) {
    dispatch(push(HOME));
    dispatch(renderFlash("error", "You do not have permissions for that page"));
    return null;
  }
  return <>{children}</>;
};

export default BasicTierRoutes;
