import Base from "fleet/base";
import Request from "fleet/request";
import accountMethods from "fleet/entities/account";
import configMethods from "fleet/entities/config";
import versionMethods from "fleet/entities/version";
import osqueryOptionsMethods from "fleet/entities/osquery_options";
import hostMethods from "fleet/entities/hosts";
import activitiesMethods from "fleet/entities/activities";
import inviteMethods from "fleet/entities/invites";
import labelMethods from "fleet/entities/labels";
import packMethods from "fleet/entities/packs";
import queryMethods from "fleet/entities/queries";
import scheduledQueryMethods from "fleet/entities/scheduled_queries";
import teamScheduledQueryMethods from "fleet/entities/team_scheduled_queries";
import globalScheduledQueryMethods from "fleet/entities/global_scheduled_queries";
import sessionMethods from "fleet/entities/sessions";
import statusLabelMethods from "fleet/entities/status_labels";
import targetMethods from "fleet/entities/targets";
import userMethods from "fleet/entities/users";
import websocketMethods from "fleet/websockets";
import statusMethods from "fleet/status";
import teamMethods from "fleet/entities/teams";

const DEFAULT_BODY = JSON.stringify({});

class Fleet extends Base {
  constructor() {
    super();

    this.account = accountMethods(this);
    this.config = configMethods(this);
    this.version = versionMethods(this);
    this.osqueryOptions = osqueryOptionsMethods(this);
    this.hosts = hostMethods(this);
    this.activities = activitiesMethods(this);
    this.invites = inviteMethods(this);
    this.labels = labelMethods(this);
    this.packs = packMethods(this);
    this.queries = queryMethods(this);
    this.scheduledQueries = scheduledQueryMethods(this);
    this.teamScheduledQueries = teamScheduledQueryMethods(this);
    this.globalScheduledQueries = globalScheduledQueryMethods(this);
    this.sessions = sessionMethods(this);
    this.statusLabels = statusLabelMethods(this);
    this.targets = targetMethods(this);
    this.users = userMethods(this);
    this.websockets = websocketMethods(this);
    this.status = statusMethods(this);
    this.teams = teamMethods(this);
  }

  authenticatedDelete(endpoint, overrideHeaders = {}, body) {
    const headers = this._authenticatedHeaders(overrideHeaders);

    return Base._deleteRequest(endpoint, headers, body);
  }

  authenticatedGet(endpoint, overrideHeaders = {}) {
    const { GET } = Request.REQUEST_METHODS;

    return this._authenticatedRequest(GET, endpoint, {}, overrideHeaders);
  }

  authenticatedPatch(endpoint, body = {}, overrideHeaders = {}) {
    const { PATCH } = Request.REQUEST_METHODS;

    return this._authenticatedRequest(PATCH, endpoint, body, overrideHeaders);
  }

  authenticatedPost(endpoint, body = DEFAULT_BODY, overrideHeaders = {}) {
    const { POST } = Request.REQUEST_METHODS;

    return this._authenticatedRequest(POST, endpoint, body, overrideHeaders);
  }

  setBearerToken(bearerToken) {
    this.bearerToken = bearerToken;
  }
}

export default new Fleet();
