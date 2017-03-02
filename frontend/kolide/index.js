import Base from 'kolide/base';
import Request from 'kolide/request';
import accountMethods from 'kolide/entities/account';
import configMethods from 'kolide/entities/config';
import configOptionMethods from 'kolide/entities/config_options';
import hostMethods from 'kolide/entities/hosts';
import inviteMethods from 'kolide/entities/invites';
import labelMethods from 'kolide/entities/labels';
import licenseMethods from 'kolide/entities/licenses';
import packMethods from 'kolide/entities/packs';
import queryMethods from 'kolide/entities/queries';
import scheduledQueryMethods from 'kolide/entities/scheduled_queries';
import sessionMethods from 'kolide/entities/sessions';
import statusLabelMethods from 'kolide/entities/status_labels';
import targetMethods from 'kolide/entities/targets';
import userMethods from 'kolide/entities/users';
import websocketMethods from 'kolide/websockets';

const DEFAULT_BODY = JSON.stringify({});

class Kolide extends Base {
  constructor () {
    super();

    this.account = accountMethods(this);
    this.config = configMethods(this);
    this.configOptions = configOptionMethods(this);
    this.hosts = hostMethods(this);
    this.invites = inviteMethods(this);
    this.labels = labelMethods(this);
    this.license = licenseMethods(this);
    this.packs = packMethods(this);
    this.queries = queryMethods(this);
    this.scheduledQueries = scheduledQueryMethods(this);
    this.sessions = sessionMethods(this);
    this.statusLabels = statusLabelMethods(this);
    this.targets = targetMethods(this);
    this.users = userMethods(this);
    this.websockets = websocketMethods(this);
  }

  getLabelHosts = () => {
    // const { LABEL_HOSTS } = endpoints;

    const stubbedResponse = {
      hosts: [
        {
          detail_updated_at: '2016-10-25T16:24:27.679472917-04:00',
          hostname: 'jmeller-mbp.local',
          id: 1,
          ip: '192.168.1.10',
          mac: '10:11:12:13:14:15',
          memory: 4145483776,
          os_version: 'Mac OS X 10.11.6',
          osquery_version: '2.0.0',
          platform: 'darwin',
          status: 'online',
          updated_at: '0001-01-01T00:00:00Z',
          uptime: 3600000000000,
          uuid: '1234-5678-9101',
        },
        {
          detail_updated_at: '2016-10-25T16:24:27.679472917-04:00',
          hostname: 'Jason Meller\'s Windows Note',
          id: 2,
          ip: '192.168.1.11',
          mac: '0C-BA-8D-45-FD-B9',
          memory: 4145483776,
          os_version: 'Windows Vista 0.0.1',
          osquery_version: '2.0.0',
          platform: 'windows',
          status: 'offline',
          updated_at: '0001-01-01T00:00:00Z',
          uptime: 3600000000000,
          uuid: '1234-5678-9101',
        },
      ],
    };

    return Promise.resolve(stubbedResponse)
      .then((response) => { return response.hosts; });
  }

  authenticatedDelete (endpoint, overrideHeaders = {}) {
    const headers = this._authenticatedHeaders(overrideHeaders);

    return Base._deleteRequest(endpoint, headers);
  }

  authenticatedGet (endpoint, overrideHeaders = {}) {
    const { GET } = Request.REQUEST_METHODS;

    return this._authenticatedRequest(GET, endpoint, {}, overrideHeaders);
  }

  authenticatedPatch (endpoint, body = {}, overrideHeaders = {}) {
    const { PATCH } = Request.REQUEST_METHODS;

    return this._authenticatedRequest(PATCH, endpoint, body, overrideHeaders);
  }

  authenticatedPost (endpoint, body = DEFAULT_BODY, overrideHeaders = {}) {
    const { POST } = Request.REQUEST_METHODS;

    return this._authenticatedRequest(POST, endpoint, body, overrideHeaders);
  }

  setBearerToken (bearerToken) {
    this.bearerToken = bearerToken;
  }
}

export default new Kolide();
