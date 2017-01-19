import Kolide from '../../../../kolide';
import reduxConfig from '../base/reduxConfig';
import schemas from '../base/schemas';

const { HOSTS: schema } = schemas;

export default reduxConfig({
  destroyFunc: Kolide.hosts.destroy,
  entityName: 'hosts',
  loadAllFunc: Kolide.hosts.loadAll,
  parseEntityFunc: (host) => {
    return { ...host, target_type: 'hosts' };
  },
  schema,
});
