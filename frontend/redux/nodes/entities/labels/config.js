import Fleet from "fleet";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { LABELS: schema } = schemas;

export default new Config({
  createFunc: Fleet.labels.create,
  destroyFunc: Fleet.labels.destroy,
  entityName: "labels",
  loadAllFunc: Fleet.labels.loadAll,
  parseEntityFunc: (label) => {
    return { ...label, target_type: "labels" };
  },
  schema,
  updateFunc: Fleet.labels.update,
});
