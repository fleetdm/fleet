import Kolide from "kolide";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { QUERIES: schema } = schemas;

export default new Config({
  createFunc: Kolide.queries.create,
  destroyFunc: Kolide.queries.destroy,
  entityName: "queries",
  loadAllFunc: Kolide.queries.loadAll,
  loadFunc: Kolide.queries.load,
  schema,
  updateFunc: Kolide.queries.update,
});
