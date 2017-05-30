import Kolide from 'kolide';
import Config from 'redux/nodes/entities/base/config';
import schemas from 'redux/nodes/entities/base/schemas';

const { DECORATORS: schema } = schemas;


export default new Config({
  entityName: 'decorators',
  loadAllFunc: Kolide.decorators.loadAll,
  createFunc: Kolide.decorators.create,
  destroyFunc: Kolide.decorators.destroy,
  updateFunc: Kolide.decorators.update,
  schema,
});
