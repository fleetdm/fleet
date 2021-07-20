import { formatGlobalScheduledQueryForClient } from "fleet/helpers";
import Fleet from "fleet";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { GLOBAL_SCHEDULED_QUERIES: schema } = schemas;

export default new Config({
  createFunc: Fleet.globalScheduledQueries.create,
  destroyFunc: Fleet.globalScheduledQueries.destroy,
  entityName: "global_scheduled_queries",
  loadAllFunc: Fleet.globalScheduledQueries.loadAll,
  parseEntityFunc: formatGlobalScheduledQueryForClient,
  schema,
  updateFunc: Fleet.globalScheduledQueries.update,
});
