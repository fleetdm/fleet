import Fleet from "fleet";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { USERS } = schemas;

export default new Config({
  createFunc: Fleet.users.create,
  destroyFunc: Fleet.users.destroy,
  entityName: "users",
  loadAllFunc: Fleet.users.loadAll,
  schema: USERS,
  updateFunc: Fleet.users.update,
});
