import Kolide from "kolide";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { PACKS: schema } = schemas;

export default new Config({
  createFunc: Kolide.packs.create,
  destroyFunc: Kolide.packs.destroy,
  entityName: "packs",
  loadAllFunc: Kolide.packs.loadAll,
  loadFunc: Kolide.packs.load,
  schema,
  updateFunc: Kolide.packs.update,
});
