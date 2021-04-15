import Kolide from "kolide";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { USERS } = schemas;

export default new Config({
  createFunc: Kolide.users.create,
  entityName: "users",
  loadAllFunc: Kolide.users.loadAll,
  schema: USERS,
  updateFunc: Kolide.users.update,
});
