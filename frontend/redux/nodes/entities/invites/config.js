import Fleet from "fleet";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { INVITES: schema } = schemas;

export default new Config({
  createFunc: Fleet.invites.create,
  destroyFunc: Fleet.invites.destroy,
  entityName: "invites",
  loadAllFunc: Fleet.invites.loadAll,
  schema,
});
