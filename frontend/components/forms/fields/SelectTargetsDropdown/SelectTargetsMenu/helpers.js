import { isEqual } from 'lodash';

export const shouldShowModal = (moreInfoTarget, target) => {
  if (!moreInfoTarget) return false;

  return isEqual(
    { id: moreInfoTarget.id, type: moreInfoTarget.target_type },
    { id: target.id, type: target.target_type },
  );
};

export default { shouldShowModal };
