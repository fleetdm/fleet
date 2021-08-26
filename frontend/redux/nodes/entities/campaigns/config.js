import { destroyFunc, update } from "redux/nodes/entities/campaigns/helpers";
import Fleet from "fleet";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { CAMPAIGNS: schema } = schemas;

export default new Config({
  createFunc: Fleet.queries.run,
  destroyFunc,
  updateFunc: update,
  entityName: "campaigns",
  schema,
});
