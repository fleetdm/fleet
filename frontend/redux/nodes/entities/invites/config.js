import Kolide from 'kolide';
import Config from 'redux/nodes/entities/base/config';
import schemas from 'redux/nodes/entities/base/schemas';

const { INVITES: schema } = schemas;

export default new Config({
  createFunc: Kolide.inviteUser,
  destroyFunc: Kolide.revokeInvite,
  entityName: 'invites',
  loadAllFunc: Kolide.getInvites,
  schema,
});
