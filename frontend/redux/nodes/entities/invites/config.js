import Kolide from 'kolide';
import reduxConfig from 'redux/nodes/entities/base/reduxConfig';
import schemas from 'redux/nodes/entities/base/schemas';

const { INVITES: schema } = schemas;

export default reduxConfig({
  createFunc: Kolide.inviteUser,
  destroyFunc: Kolide.revokeInvite,
  entityName: 'invites',
  loadAllFunc: Kolide.getInvites,
  schema,
});
