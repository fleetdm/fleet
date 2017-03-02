import Kolide from 'kolide';
import Config from 'redux/nodes/entities/base/config';
import schemas from 'redux/nodes/entities/base/schemas';

const { PACKS: schema } = schemas;

export default new Config({
  createFunc: Kolide.createPack,
  destroyFunc: Kolide.destroyPack,
  entityName: 'packs',
  loadAllFunc: Kolide.getPacks,
  loadFunc: Kolide.getPack,
  schema,
  updateFunc: Kolide.updatePack,
});
