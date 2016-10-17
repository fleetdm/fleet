import { Schema } from 'normalizr';

const invitesSchema = new Schema('invites');
const usersSchema = new Schema('users');
const hostsSchema = new Schema('hosts');

export default {
  HOSTS: hostsSchema,
  INVITES: invitesSchema,
  USERS: usersSchema,
};
