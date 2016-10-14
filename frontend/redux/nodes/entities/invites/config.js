import { addGravatarUrlToResource } from '../base/helpers';
import Kolide from '../../../../kolide';
import reduxConfig from '../base/reduxConfig';
import schemas from '../base/schemas';

const { INVITES: schema } = schemas;

export default reduxConfig({
  createFunc: Kolide.inviteUser,
  destroyFunc: Kolide.revokeInvite,
  entityName: 'invites',
  loadAllFunc: Kolide.getInvites,
  parseFunc: addGravatarUrlToResource,
  schema,
});

