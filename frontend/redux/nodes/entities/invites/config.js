import Kolide from "kolide";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { INVITES: schema } = schemas;

export default new Config({
  createFunc: Kolide.invites.create,
  destroyFunc: Kolide.invites.destroy,
  entityName: "invites",
  loadAllFunc: Kolide.invites.loadAll,
  schema,
});
