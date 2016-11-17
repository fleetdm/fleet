import { Schema } from 'normalizr';

const hostsSchema = new Schema('hosts');
const invitesSchema = new Schema('invites');
const labelsSchema = new Schema('labels');
const queriesSchema = new Schema('queries');
const targetsSchema = new Schema('targets');
const usersSchema = new Schema('users');

export default {
  HOSTS: hostsSchema,
  INVITES: invitesSchema,
  LABELS: labelsSchema,
  QUERIES: queriesSchema,
  TARGETS: targetsSchema,
  USERS: usersSchema,
};
