import { formatTeamScheduledQueryForClient } from "fleet/helpers";
import Fleet from "fleet";
import Config from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";

const { TEAM_SCHEDULED_QUERIES: schema } = schemas;

export default new Config({
  createFunc: Fleet.teamScheduledQueries.create,
  destroyFunc: Fleet.teamScheduledQueries.destroy,
  entityName: "team_scheduled_queries",
  loadAllFunc: Fleet.teamScheduledQueries.loadAll,
  parseEntityFunc: formatTeamScheduledQueryForClient,
  schema,
  updateFunc: Fleet.teamScheduledQueries.update,
});
