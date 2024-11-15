import fleetAppData from "../../../../../../server/mdm/maintainedapps/apps.json";

const getFleetAppData = (id: number) => {
  return fleetAppData[id];
};

export const getFleetAppPolicyName = (appName: string) => {
  return `[Install software] ${appName}`;
};

export const getFleetAppPolicyDescription = (appName: string) => {
  return `"Policy triggers automatic install of ${appName} on each host that's missing this software."`;
};

export const getFleetAppPolicyQuery = (id: number) => {
  const app = getFleetAppData(id); // TODO: need a better matching mechanism here
  return app.automatic_policy_query;
};
