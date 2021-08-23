import { Dispatch } from "redux";
import { push } from "react-router-redux";
import PATHS from "router/paths";
import { differenceWith, isEqual, uniqWith } from "lodash";

import permissionUtils from "utilities/permissions";
import hostActions from "redux/nodes/entities/hosts/actions"; // @ts-ignore
import { setSelectedTargets } from "redux/nodes/components/QueryPages/actions";
import { IHost } from "interfaces/host";
import { ITarget } from "interfaces/target";
import { IQuery } from "interfaces/query";
import { IUser } from "interfaces/user";

const targetsChanged = (hosts: IHost[], targets: ITarget[]) => {
  const sameLength = hosts.length === targets.length;

  if (sameLength) {
    const delta = differenceWith(hosts, targets, isEqual);

    return !!delta.length;
  }

  return true;
};

const comparator = (arrayVal: any, otherVal: any) => {
  return (
    arrayVal.target_type === otherVal.target_type && arrayVal.id === otherVal.id
  );
};

export const selectHosts = (
  dispatch: Dispatch,
  { hosts = [], selectedTargets = [] }
) => {
  if (!hosts.length || !targetsChanged(hosts, selectedTargets)) {
    return false;
  }

  const newTargets = uniqWith([...hosts, ...selectedTargets], comparator);

  dispatch(setSelectedTargets(newTargets));

  return false;
};

export const fetchHost = (dispatch: Dispatch, hostID: string) => {
  return dispatch(hostActions.load(hostID)).catch(() => {
    dispatch(push(PATHS.FLEET_500));

    return false;
  });
};

export const showDropdown = (query: IQuery, currentUser: IUser) => {
  if (query.observer_can_run) {
    return true;
  }
  return !permissionUtils.isOnlyObserver(currentUser);
};

export const hasSavePermissions = (currentUser: IUser) => {
  return (
    permissionUtils.isGlobalAdmin(currentUser) ||
    permissionUtils.isGlobalMaintainer(currentUser)
  );
};

export default { selectHosts, fetchHost, showDropdown, hasSavePermissions };
