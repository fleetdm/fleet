import { flatMap } from 'lodash';

const filterTarget = (targetType) => {
  return (target) => {
    return target.target_type === targetType ? [target.id] : [];
  };
};

export const formatSelectedTargetsForApi = (selectedTargets) => {
  const targets = selectedTargets || [];
  const hosts = flatMap(targets, filterTarget('hosts'));
  const labels = flatMap(targets, filterTarget('labels'));

  return { hosts, labels };
};

export default { formatSelectedTargetsForApi };
