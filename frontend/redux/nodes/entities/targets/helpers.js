import { map } from 'lodash';

export const appendTargetTypeToTargets = (apiResponse) => {
  const { targets } = apiResponse;
  const hosts = map(targets.hosts, (host) => {
    return {
      ...host,
      target_type: 'hosts',
    };
  });
  const labels = map(targets.labels, (label) => {
    return {
      ...label,
      target_type: 'labels',
    };
  });

  return {
    ...apiResponse,
    targets: [
      ...hosts,
      ...labels,
    ],
  };
};

export default { appendTargetTypeToTargets };
