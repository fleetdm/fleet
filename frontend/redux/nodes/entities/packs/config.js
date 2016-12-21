import Kolide from '../../../../kolide';
import reduxConfig from '../base/reduxConfig';
import schemas from '../base/schemas';

const { PACKS: schema } = schemas;

export default reduxConfig({
  createFunc: Kolide.createPack,
  entityName: 'packs',
  loadAllFunc: Kolide.getPacks,
  loadFunc: Kolide.getPack,
  schema,
});
