import { combineReducers } from "redux";

import campaigns from "./campaigns/reducer";
import hosts from "./hosts/reducer";
import invites from "./invites/reducer";
import labels from "./labels/reducer";
import packs from "./packs/reducer";
import queries from "./queries/reducer";
import globalScheduledQueries from "./global_scheduled_queries/reducer";
import teamScheduledQueries from "./team_scheduled_queries/reducer";
import scheduledQueries from "./scheduled_queries/reducer";
import users from "./users/reducer";
import teams from "./teams/reducer";

export default combineReducers({
  campaigns,
  hosts,
  invites,
  labels,
  packs,
  queries,
  global_scheduled_queries: globalScheduledQueries,
  team_scheduled_queries: teamScheduledQueries,
  scheduled_queries: scheduledQueries,
  users,
  teams,
});
