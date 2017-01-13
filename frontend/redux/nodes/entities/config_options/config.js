import Kolide from 'kolide';
import reduxConfig from 'redux/nodes/entities/base/reduxConfig';
import schemas from 'redux/nodes/entities/base/schemas';

const { CONFIG_OPTIONS: schema } = schemas;

export default reduxConfig({
  entityName: 'config_options',
  loadAllFunc: Kolide.configOptions.loadAll,
  schema,
  updateFunc: Kolide.configOptions.update,
});

