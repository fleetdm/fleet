import { Schema } from 'normalizr';

const hostsSchema = new Schema('hosts');
const invitesSchema = new Schema('invites');
const targetsSchema = new Schema('targets');
const usersSchema = new Schema('users');

export default {
  HOSTS: hostsSchema,
  INVITES: invitesSchema,
  TARGETS: targetsSchema,
  USERS: usersSchema,
};
