import Fleet from "fleet";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { QUERIES: schema } = schemas;

export default new Config({
  createFunc: Fleet.queries.create,
  destroyFunc: Fleet.queries.destroy,
  entityName: "queries",
  loadAllFunc: Fleet.queries.loadAll,
  loadFunc: Fleet.queries.load,
  schema,
  updateFunc: Fleet.queries.update,
});
