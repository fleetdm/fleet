import Kolide from 'kolide';
import reduxConfig from 'redux/nodes/entities/base/reduxConfig';
import schemas from 'redux/nodes/entities/base/schemas';
import { parseEntityFunc } from 'redux/nodes/entities/hosts/helpers';

const { HOSTS: schema } = schemas;

export default reduxConfig({
  destroyFunc: Kolide.hosts.destroy,
  entityName: 'hosts',
  loadAllFunc: Kolide.hosts.loadAll,
  parseEntityFunc,
  schema,
});
