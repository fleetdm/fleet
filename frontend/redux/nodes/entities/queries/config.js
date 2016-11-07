import Kolide from '../../../../kolide';
import reduxConfig from '../base/reduxConfig';
import schemas from '../base/schemas';

const { QUERIES: schema } = schemas;

export default reduxConfig({
  createFunc: Kolide.createQuery,
  entityName: 'queries',
  loadFunc: Kolide.getQuery,
  schema,
});
