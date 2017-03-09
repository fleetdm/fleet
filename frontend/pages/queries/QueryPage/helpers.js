import { differenceWith, isEqual, uniqWith } from 'lodash';

import { setSelectedTargets } from 'redux/nodes/components/QueryPages/actions';

const targetsChanged = (hosts, targets) => {
  const sameLength = hosts.length === targets.length;

  if (sameLength) {
    const delta = differenceWith(hosts, targets, isEqual);

    return !!delta.length;
  }

  return true;
};

const comparator = (arrayVal, otherVal) => {
  return arrayVal.target_type === otherVal.target_type &&
    arrayVal.id === otherVal.id;
};

const selectHosts = (dispatch, { hosts = [], selectedTargets = [] }) => {
  if (!hosts.length || !targetsChanged(hosts, selectedTargets)) {
    return false;
  }

  const newTargets = uniqWith([...hosts, ...selectedTargets], comparator);

  dispatch(setSelectedTargets(newTargets));

  return false;
};

export default { selectHosts };
