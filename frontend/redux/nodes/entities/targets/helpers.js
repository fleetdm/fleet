import { map } from 'lodash';

export const appendTargetTypeToTargets = (targets, targetType) => {
  return map(targets, (target) => {
    return {
      ...target,
      target_type: targetType,
    };
  });
};

export default { appendTargetTypeToTargets };
