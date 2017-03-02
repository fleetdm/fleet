import Kolide from 'kolide';
import Config from 'redux/nodes/entities/base/config';
import schemas from 'redux/nodes/entities/base/schemas';

const { USERS } = schemas;

export default new Config({
  createFunc: Kolide.createUser,
  entityName: 'users',
  loadAllFunc: Kolide.getUsers,
  schema: USERS,
  updateFunc: Kolide.updateUser,
});
