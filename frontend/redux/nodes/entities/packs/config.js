import Kolide from 'kolide';
import reduxConfig from 'redux/nodes/entities/base/reduxConfig';
import schemas from 'redux/nodes/entities/base/schemas';

const { PACKS: schema } = schemas;

export default reduxConfig({
  createFunc: Kolide.createPack,
  destroyFunc: Kolide.destroyPack,
  entityName: 'packs',
  loadAllFunc: Kolide.getPacks,
  loadFunc: Kolide.getPack,
  schema,
  updateFunc: Kolide.updatePack,
});
