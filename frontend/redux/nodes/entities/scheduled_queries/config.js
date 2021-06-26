import { formatScheduledQueryForClient } from "fleet/helpers";
import Fleet from "fleet";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { SCHEDULED_QUERIES: schema } = schemas;

export default new Config({
  createFunc: Fleet.scheduledQueries.create,
  destroyFunc: Fleet.scheduledQueries.destroy,
  entityName: "scheduled_queries",
  loadAllFunc: Fleet.scheduledQueries.loadAll,
  parseEntityFunc: formatScheduledQueryForClient,
  schema,
  updateFunc: Fleet.scheduledQueries.update,
});
