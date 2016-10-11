import Kolide from '../../../../kolide';
import reduxConfig from '../base/reduxConfig';
import schemas from '../base/schemas';

const { INVITES: schema } = schemas;

export default reduxConfig({
  createFunc: Kolide.inviteUser,
  entityName: 'invites',
  schema,
});

