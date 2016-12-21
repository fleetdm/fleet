import Kolide from 'kolide';
import reduxConfig from 'redux/nodes/entities/base/reduxConfig';
import schemas from 'redux/nodes/entities/base/schemas';

const { SCHEDULED_QUERIES: schema } = schemas;

export default reduxConfig({
  createFunc: Kolide.createScheduledQuery,
  destroyFunc: Kolide.destroyScheduledQuery,
  entityName: 'scheduled_queries',
  loadAllFunc: Kolide.getScheduledQueries,
  schema,
});

