import Base from "kolide/base";
import Request from "kolide/request";
import accountMethods from "kolide/entities/account";
import configMethods from "kolide/entities/config";
import versionMethods from "kolide/entities/version";
import osqueryOptionsMethods from "kolide/entities/osquery_options";
import hostMethods from "kolide/entities/hosts";
import inviteMethods from "kolide/entities/invites";
import labelMethods from "kolide/entities/labels";
import packMethods from "kolide/entities/packs";
import queryMethods from "kolide/entities/queries";
import scheduledQueryMethods from "kolide/entities/scheduled_queries";
import sessionMethods from "kolide/entities/sessions";
import statusLabelMethods from "kolide/entities/status_labels";
import targetMethods from "kolide/entities/targets";
import userMethods from "kolide/entities/users";
import websocketMethods from "kolide/websockets";
import statusMethods from "kolide/status";

const DEFAULT_BODY = JSON.stringify({});

class Kolide extends Base {
  constructor() {
    super();

    this.account = accountMethods(this);
    this.config = configMethods(this);
    this.version = versionMethods(this);
    this.osqueryOptions = osqueryOptionsMethods(this);
    this.hosts = hostMethods(this);
    this.invites = inviteMethods(this);
    this.labels = labelMethods(this);
    this.packs = packMethods(this);
    this.queries = queryMethods(this);
    this.scheduledQueries = scheduledQueryMethods(this);
    this.sessions = sessionMethods(this);
    this.statusLabels = statusLabelMethods(this);
    this.targets = targetMethods(this);
    this.users = userMethods(this);
    this.websockets = websocketMethods(this);
    this.status = statusMethods(this);
  }

  authenticatedDelete(endpoint, overrideHeaders = {}) {
    const headers = this._authenticatedHeaders(overrideHeaders);

    return Base._deleteRequest(endpoint, headers);
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

export default new Kolide();
