import { IHost } from "../interfaces/host";
import { IPack } from "../interfaces/pack";
import { IQuery } from "../interfaces/query";
import URL_PREFIX from "./url_prefix";

export default {
  ADMIN_USERS: `${URL_PREFIX}/settings/users`,
  ADMIN_SETTINGS: `${URL_PREFIX}/settings/organization`,
  ADMIN_OSQUERY: `${URL_PREFIX}/settings/osquery`,
  ALL_PACKS: `${URL_PREFIX}/packs/all`,
  EDIT_PACK: (pack: IPack): string => {
    return `${URL_PREFIX}/packs/${pack.id}/edit`;
  },
  PACK: (pack: IPack): string => {
    return `${URL_PREFIX}/packs/${pack.id}`;
  },
  EDIT_QUERY: (query: IQuery): string => {
    return `${URL_PREFIX}/queries/${query.id}`;
  },
  FORGOT_PASSWORD: `${URL_PREFIX}/login/forgot`,
  HOME: `${URL_PREFIX}/`,
  FLEET_500: `${URL_PREFIX}/500`,
  LOGIN: `${URL_PREFIX}/login`,
  LOGOUT: `${URL_PREFIX}/logout`,
  MANAGE_HOSTS: `${URL_PREFIX}/hosts/manage`,
  HOST_DETAILS: (host: IHost): string => {
    return `${URL_PREFIX}/hosts/${host.id}`;
  },
  MANAGE_PACKS: `${URL_PREFIX}/packs/manage`,
  NEW_PACK: `${URL_PREFIX}/packs/new`,
  MANAGE_QUERIES: `${URL_PREFIX}/queries/manage`,
  NEW_QUERY: `${URL_PREFIX}/queries/new`,
  RESET_PASSWORD: `${URL_PREFIX}/login/reset`,
  SETUP: `${URL_PREFIX}/setup`,
  USER_SETTINGS: `${URL_PREFIX}/profile`,
};
