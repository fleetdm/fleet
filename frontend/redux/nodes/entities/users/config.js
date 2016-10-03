import md5 from 'js-md5';
import Kolide from '../../../../kolide';
import reduxConfig from '../base/reduxConfig';
import schemas from '../base/schemas';

const { USERS } = schemas;

export default reduxConfig({
  entityName: 'users',
  loadFunc: Kolide.getUsers,
  parseFunc: (user) => {
    const { email } = user;
    const emailHash = md5(email.toLowerCase());
    const gravatarURL = `https://www.gravatar.com/avatar/${emailHash}`;

    return {
      ...user,
      gravatarURL,
    };
  },
  schema: USERS,
  updateFunc: Kolide.updateUser,
});
