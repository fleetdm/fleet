import Kolide from 'kolide';
import Config from 'redux/nodes/entities/base/config';
import schemas from 'redux/nodes/entities/base/schemas';

const { CONFIG_OPTIONS: schema } = schemas;

export default new Config({
  entityName: 'config_options',
  loadAllFunc: Kolide.configOptions.loadAll,
  schema,
  updateFunc: Kolide.configOptions.update,
});

