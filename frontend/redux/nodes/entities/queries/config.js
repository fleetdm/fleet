import Kolide from 'kolide';
import Config from 'redux/nodes/entities/base/config';
import schemas from 'redux/nodes/entities/base/schemas';

const { QUERIES: schema } = schemas;

export default new Config({
  createFunc: Kolide.createQuery,
  destroyFunc: Kolide.destroyQuery,
  entityName: 'queries',
  loadAllFunc: Kolide.getQueries,
  loadFunc: Kolide.getQuery,
  schema,
  updateFunc: Kolide.updateQuery,
});
