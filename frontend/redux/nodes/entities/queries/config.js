import Kolide from 'kolide';
import reduxConfig from 'redux/nodes/entities/base/reduxConfig';
import schemas from 'redux/nodes/entities/base/schemas';

const { QUERIES: schema } = schemas;

export default reduxConfig({
  createFunc: Kolide.createQuery,
  destroyFunc: Kolide.destroyQuery,
  entityName: 'queries',
  loadAllFunc: Kolide.getQueries,
  loadFunc: Kolide.getQuery,
  schema,
  updateFunc: Kolide.updateQuery,
});
