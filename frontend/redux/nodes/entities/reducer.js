import { combineReducers } from "redux";

import campaigns from "./campaigns/reducer";
import hosts from "./hosts/reducer";
import invites from "./invites/reducer";
import labels from "./labels/reducer";
import packs from "./packs/reducer";
import queries from "./queries/reducer";
import scheduledQueries from "./scheduled_queries/reducer";
import users from "./users/reducer";

export default combineReducers({
  campaigns,
  hosts,
  invites,
  labels,
  packs,
  queries,
  scheduled_queries: scheduledQueries,
  users,
});
