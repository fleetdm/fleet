import { addGravatarUrlToResource } from '../base/helpers';
import Kolide from '../../../../kolide';
import reduxConfig from '../base/reduxConfig';
import schemas from '../base/schemas';

const { USERS } = schemas;

export default reduxConfig({
  entityName: 'users',
  loadAllFunc: Kolide.getUsers,
  parseFunc: addGravatarUrlToResource,
  schema: USERS,
  updateFunc: Kolide.updateUser,
});
