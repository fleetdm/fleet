import Kolide from "kolide";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { LABELS: schema } = schemas;

export default new Config({
  createFunc: Kolide.labels.create,
  destroyFunc: Kolide.labels.destroy,
  entityName: "labels",
  loadAllFunc: Kolide.labels.loadAll,
  parseEntityFunc: (label) => {
    return { ...label, target_type: "labels" };
  },
  schema,
  updateFunc: Kolide.labels.update,
});
