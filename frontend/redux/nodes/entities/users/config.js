import Kolide from 'kolide';
import reduxConfig from 'redux/nodes/entities/base/reduxConfig';
import schemas from 'redux/nodes/entities/base/schemas';

const { USERS } = schemas;

export default reduxConfig({
  entityName: 'users',
  loadAllFunc: Kolide.getUsers,
  schema: USERS,
  updateFunc: Kolide.updateUser,
});
