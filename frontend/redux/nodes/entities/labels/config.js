import Kolide from '../../../../kolide';
import reduxConfig from '../base/reduxConfig';
import schemas from '../base/schemas';

const { LABELS: schema } = schemas;

export default reduxConfig({
  createFunc: Kolide.labels.create,
  destroyFunc: Kolide.labels.destroy,
  entityName: 'labels',
  loadAllFunc: Kolide.labels.loadAll,
  parseEntityFunc: (label) => {
    return { ...label, target_type: 'labels' };
  },
  schema,
  updateFunc: Kolide.labels.update,
});
