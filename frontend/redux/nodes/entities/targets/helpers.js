import { map } from "lodash";

import { parseEntityFunc } from "redux/nodes/entities/hosts/helpers";

export const appendTargetTypeToTargets = (targets, targetType) => {
  return map(targets, (target) => {
    if (targetType === "hosts") {
      return parseEntityFunc(target);
    }
    // added 5/26 this is wrong, look into this
    if (targetType === "teams") {
      return parseEntityFunc(target);
    }

    return {
      ...target,
      target_type: targetType,
    };
  });
};

export default { appendTargetTypeToTargets };
