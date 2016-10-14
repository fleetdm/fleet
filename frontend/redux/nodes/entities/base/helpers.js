import md5 from 'js-md5';
import { pickBy } from 'lodash';

export const addGravatarUrlToResource = (resource) => {
  const { email } = resource;

  const emailHash = md5(email.toLowerCase());
  const gravatarURL = `https://www.gravatar.com/avatar/${emailHash}`;

  return {
    ...resource,
    gravatarURL,
  };
};

export const entitiesExceptID = (entities, id) => {
  return pickBy(entities, (entity, key) => {
    return String(key) !== String(id);
  });
};

export default { entitiesExceptID, addGravatarUrlToResource };
