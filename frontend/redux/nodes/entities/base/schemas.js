import { Schema } from "normalizr";

const campaignsSchema = new Schema("campaigns");
const hostsSchema = new Schema("hosts");
const invitesSchema = new Schema("invites");
const labelsSchema = new Schema("labels");
const packsSchema = new Schema("packs");
const queriesSchema = new Schema("queries");
const globalScheduledQueriesSchema = new Schema("global_scheduled_queries");
const teamScheduledQueriesSchema = new Schema("team_scheduled_queries");
const scheduledQueriesSchema = new Schema("scheduled_queries");
const targetsSchema = new Schema("targets");
const usersSchema = new Schema("users");
const teamsSchema = new Schema("teams");

export default {
  CAMPAIGNS: campaignsSchema,
  HOSTS: hostsSchema,
  INVITES: invitesSchema,
  LABELS: labelsSchema,
  PACKS: packsSchema,
  QUERIES: queriesSchema,
  GLOBAL_SCHEDULED_QUERIES: globalScheduledQueriesSchema,
  TEAM_SCHEDULED_QUERIES: teamScheduledQueriesSchema,
  SCHEDULED_QUERIES: scheduledQueriesSchema,
  TARGETS: targetsSchema,
  USERS: usersSchema,
  TEAMS: teamsSchema,
};
