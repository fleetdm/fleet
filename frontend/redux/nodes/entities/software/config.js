import Fleet from "fleet";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { SOFTWARE } = schemas;

export default new Config({
  entityName: "software",
  // loadAllFunc: Fleet.software.loadAll,
  schema: SOFTWARE,
});
