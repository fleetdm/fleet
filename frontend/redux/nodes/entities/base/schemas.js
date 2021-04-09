import { Schema } from "normalizr";

const campaignsSchema = new Schema("campaigns");
const hostsSchema = new Schema("hosts");
const invitesSchema = new Schema("invites");
const labelsSchema = new Schema("labels");
const packsSchema = new Schema("packs");
const queriesSchema = new Schema("queries");
const scheduledQueriesSchema = new Schema("scheduled_queries");
const targetsSchema = new Schema("targets");
const usersSchema = new Schema("users");

export default {
  CAMPAIGNS: campaignsSchema,
  HOSTS: hostsSchema,
  INVITES: invitesSchema,
  LABELS: labelsSchema,
  PACKS: packsSchema,
  QUERIES: queriesSchema,
  SCHEDULED_QUERIES: scheduledQueriesSchema,
  TARGETS: targetsSchema,
  USERS: usersSchema,
};
