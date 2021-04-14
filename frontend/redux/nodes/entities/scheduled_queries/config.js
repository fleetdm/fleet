import helpers from "kolide/helpers";
import Kolide from "kolide";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { SCHEDULED_QUERIES: schema } = schemas;

export default new Config({
  createFunc: Kolide.scheduledQueries.create,
  destroyFunc: Kolide.scheduledQueries.destroy,
  entityName: "scheduled_queries",
  loadAllFunc: Kolide.scheduledQueries.loadAll,
  parseEntityFunc: helpers.formatScheduledQueryForClient,
  schema,
  updateFunc: Kolide.scheduledQueries.update,
});
