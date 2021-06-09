import { push } from "react-router-redux";
import PATHS from "router/paths";
import { differenceWith, isEqual, uniqWith } from "lodash";

import permissionUtils from "utilities/permissions";
import hostActions from "redux/nodes/entities/hosts/actions";
import { setSelectedTargets } from "redux/nodes/components/QueryPages/actions";

const targetsChanged = (hosts, targets) => {
  const sameLength = hosts.length === targets.length;

  if (sameLength) {
    const delta = differenceWith(hosts, targets, isEqual);

    return !!delta.length;
  }

  return true;
};

const comparator = (arrayVal, otherVal) => {
  return (
    arrayVal.target_type === otherVal.target_type && arrayVal.id === otherVal.id
  );
};

const selectHosts = (dispatch, { hosts = [], selectedTargets = [] }) => {
  if (!hosts.length || !targetsChanged(hosts, selectedTargets)) {
    return false;
  }

  const newTargets = uniqWith([...hosts, ...selectedTargets], comparator);

  dispatch(setSelectedTargets(newTargets));

  return false;
};

// TODO: pull out to common module. This same code is used in HostDetailsPage/helpers.js
export const fetchHost = (dispatch, hostID) => {
  return dispatch(hostActions.load(hostID)).catch(() => {
    dispatch(push(PATHS.FLEET_500));

    return false;
  });
};

export const showDropdown = (query, currentUser) => {
  if (query.observer_can_run) {
    return true;
  }
  return !permissionUtils.isOnlyObserver(currentUser);
};

export const hasSavePermissions = (currentUser) => {
  return (
    permissionUtils.isGlobalAdmin(currentUser) ||
    permissionUtils.isGlobalMaintainer(currentUser)
  );
};

export default { selectHosts, fetchHost, showDropdown, hasSavePermissions };
