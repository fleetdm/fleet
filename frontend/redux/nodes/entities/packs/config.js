import Fleet from "fleet";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { PACKS: schema } = schemas;

export default new Config({
  createFunc: Fleet.packs.create,
  destroyFunc: Fleet.packs.destroy,
  entityName: "packs",
  loadAllFunc: Fleet.packs.loadAll,
  loadFunc: Fleet.packs.load,
  schema,
  updateFunc: Fleet.packs.update,
});
