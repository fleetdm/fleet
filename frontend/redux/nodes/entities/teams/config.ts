// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Fleet from "fleet";
// @ts-ignore
import Config from "redux/nodes/entities/base/config";
// @ts-ignore
import schemas from "redux/nodes/entities/base/schemas";
import { formatTeamForClient } from "fleet/helpers";

const { TEAMS } = schemas;

export default new Config({
  createFunc: Fleet.teams.create,
  destroyFunc: Fleet.teams.destroy,
  entityName: "teams",
  loadFunc: Fleet.teams.load,
  loadAllFunc: Fleet.teams.loadAll,
  parseEntityFunc: formatTeamForClient,
  schema: TEAMS,
  updateFunc: Fleet.teams.update,
});
