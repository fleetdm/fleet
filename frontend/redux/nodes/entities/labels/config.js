import Kolide from '../../../../kolide';
import reduxConfig from '../base/reduxConfig';
import schemas from '../base/schemas';

const { LABELS: schema } = schemas;

export default reduxConfig({
  createFunc: Kolide.createLabel,
  entityName: 'labels',
  loadAllFunc: Kolide.getLabels,
  parseEntityFunc: (label) => {
    return { ...label, target_type: 'labels' };
  },
  schema,
});
