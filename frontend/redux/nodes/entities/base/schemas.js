import { Schema } from 'normalizr';

const invitesSchema = new Schema('invites');
const usersSchema = new Schema('users');

export default {
  INVITES: invitesSchema,
  USERS: usersSchema,
};
