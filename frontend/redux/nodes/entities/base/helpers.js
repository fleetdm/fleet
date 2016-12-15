import { pickBy } from 'lodash';

export const entitiesExceptID = (entities, id) => {
  return pickBy(entities, (entity, key) => {
    return String(key) !== String(id);
  });
};

export default { entitiesExceptID };
