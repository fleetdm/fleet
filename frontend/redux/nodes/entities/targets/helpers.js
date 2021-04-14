import { map } from "lodash";

import { parseEntityFunc } from "redux/nodes/entities/hosts/helpers";

export const appendTargetTypeToTargets = (targets, targetType) => {
  return map(targets, (target) => {
    if (targetType === "hosts") {
      return parseEntityFunc(target);
    }

    return {
      ...target,
      target_type: targetType,
    };
  });
};

export default { appendTargetTypeToTargets };
