import { get } from 'lodash';

import { appendTargetTypeToTargets } from 'redux/nodes/entities/targets/helpers';
import Base from 'kolide/base';
import endpoints from 'kolide/endpoints';
import helpers from 'kolide/helpers';
import local from 'utilities/local';

class Kolide extends Base {
  addLabelToPack = (packID, labelID) => {
    const path = `/v1/kolide/packs/${packID}/labels/${labelID}`;

    return this.authenticatedPost(this.endpoint(path));
  }

  addQueryToPack = ({ packID, queryID }) => {
    const endpoint = `/v1/kolide/packs/${packID}/queries/${queryID}`;

    return this.authenticatedPost(this.endpoint(endpoint));
  }

  configOptions = {
    loadAll: () => {
      const { CONFIG_OPTIONS } = endpoints;

      return this.authenticatedGet(this.endpoint(CONFIG_OPTIONS))
        .then(response => response.options);
    },
    update: (options) => {
      const { CONFIG_OPTIONS } = endpoints;

      return this.authenticatedPatch(this.endpoint(CONFIG_OPTIONS), JSON.stringify({ options }))
        .then(response => response.options);
    },
  }

  statusLabels = {
    getCounts: () => {
      const { STATUS_LABEL_COUNTS } = endpoints;

      return this.authenticatedGet(this.endpoint(STATUS_LABEL_COUNTS));
    },
  }

  createLabel = ({ description, name, query }) => {
    const { LABELS } = endpoints;

    return this.authenticatedPost(this.endpoint(LABELS), JSON.stringify({ description, name, query }))
      .then((response) => {
        const { label } = response;

        return {
          ...label,
          slug: helpers.labelSlug(label),
          type: 'custom',
        };
      });
  }

  createPack = ({ name, description, targets }) => {
    const { PACKS } = endpoints;
    const packTargets = helpers.formatSelectedTargetsForApi(targets, true);

    return this.authenticatedPost(this.endpoint(PACKS), JSON.stringify({ description, name, ...packTargets }))
      .then((response) => { return response.pack; });
  }

  createQuery = ({ description, name, query }) => {
    const { QUERIES } = endpoints;

    return this.authenticatedPost(this.endpoint(QUERIES), JSON.stringify({ description, name, query }))
      .then((response) => { return response.query; });
  }

  createScheduledQuery = ({ interval, logging_type: loggingType, pack_id: packID, platform, query_id: queryID, version }) => {
    const removed = loggingType === 'differential';
    const snapshot = loggingType === 'snapshot';

    const formData = {
      interval: Number(interval),
      pack_id: Number(packID),
      platform,
      query_id: Number(queryID),
      removed,
      snapshot,
      version,
    };

    return this.authenticatedPost(this.endpoint('/v1/kolide/schedule'), JSON.stringify(formData))
      .then(response => response.scheduled);
  }

  destroyQuery = ({ id }) => {
    const { QUERIES } = endpoints;
    const endpoint = `${this.endpoint(QUERIES)}/${id}`;

    return this.authenticatedDelete(endpoint);
  }

  destroyPack = ({ id }) => {
    const { PACKS } = endpoints;
    const endpoint = `${this.endpoint(PACKS)}/${id}`;

    return this.authenticatedDelete(endpoint);
  }

  destroyScheduledQuery = ({ id }) => {
    const endpoint = `${this.endpoint('/v1/kolide/schedule')}/${id}`;

    return this.authenticatedDelete(endpoint);
  }

  createUser = (formData) => {
    const { USERS } = endpoints;

    return this.authenticatedPost(this.endpoint(USERS), JSON.stringify(formData))
      .then((response) => { return response.user; });
  }

  forgotPassword ({ email }) {
    const { FORGOT_PASSWORD } = endpoints;
    const forgotPasswordEndpoint = this.baseURL + FORGOT_PASSWORD;

    return Base.post(forgotPasswordEndpoint, JSON.stringify({ email }));
  }

  getConfig = () => {
    const { CONFIG } = endpoints;

    return this.authenticatedGet(this.endpoint(CONFIG));
  }

  getInvites = () => {
    const { INVITES } = endpoints;

    return this.authenticatedGet(this.endpoint(INVITES))
      .then((response) => {
        const { invites } = response;

        return invites.map((invite) => {
          return helpers.addGravatarUrlToResource(invite);
        });
      });
  }

  getHosts = () => {
    const { HOSTS } = endpoints;

    return this.authenticatedGet(this.endpoint(HOSTS))
      .then(response => response.hosts);
  }

  getLabelHosts = (labelID) => {
    const { LABEL_HOSTS } = endpoints;
    console.log(LABEL_HOSTS(labelID));

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

  getPack = (packID) => {
    const { PACKS } = endpoints;
    const getPackEndpoint = `${this.baseURL}${PACKS}/${packID}`;

    return this.authenticatedGet(getPackEndpoint)
      .then((response) => { return response.pack; });
  }

  getQuery = (queryID) => {
    const { QUERIES } = endpoints;
    const getQueryEndpoint = `${this.baseURL}${QUERIES}/${queryID}`;

    return this.authenticatedGet(getQueryEndpoint)
      .then((response) => { return response.query; });
  }

  getQueries = () => {
    const { QUERIES } = endpoints;

    return this.authenticatedGet(this.endpoint(QUERIES))
      .then((response) => { return response.queries; });
  }

  getTargets = (query, selected = { hosts: [], labels: [] }) => {
    const { TARGETS } = endpoints;

    return this.authenticatedPost(this.endpoint(TARGETS), JSON.stringify({ query, selected }))
      .then((response) => {
        const { targets } = response;

        return {
          ...response,
          targets: [
            ...appendTargetTypeToTargets(targets.hosts, 'hosts'),
            ...appendTargetTypeToTargets(targets.labels, 'labels'),
          ],
        };
      });
  }

  getLabels = () => {
    const { LABELS } = endpoints;

    return this.authenticatedGet(this.endpoint(LABELS))
      .then((response) => {
        const labelTypeForDisplayText = {
          'All Hosts': 'all',
          'MS Windows': 'platform',
          'CentOS Linux': 'platform',
          'Mac OS X': 'platform',
          'Ubuntu Linux': 'platform',
        };
        const labels = response.labels.map((label) => {
          return {
            ...label,
            slug: helpers.labelSlug(label),
            type: labelTypeForDisplayText[label.display_text] || 'custom',
          };
        });
        const stubbedLabels = [
          { id: 'online', display_text: 'ONLINE', slug: 'online', type: 'status', count: 0 },
          { id: 'offline', display_text: 'OFFLINE', slug: 'offline', type: 'status', count: 0 },
          { id: 'mia', display_text: 'MIA', description: '(offline > 30 days)', slug: 'mia', type: 'status', count: 0 },
        ];

        return labels.concat(stubbedLabels);
      });
  }

  getPacks = () => {
    const { PACKS } = endpoints;

    return this.authenticatedGet(this.endpoint(PACKS))
      .then((response) => { return response.packs; });
  }

  getScheduledQueries = (pack) => {
    const { SCHEDULED_QUERIES } = endpoints;
    const scheduledQueryPath = SCHEDULED_QUERIES(pack);

    return this.authenticatedGet(this.endpoint(scheduledQueryPath))
      .then(response => response.scheduled);
  }

  getUsers = () => {
    const { USERS } = endpoints;

    return this.authenticatedGet(this.endpoint(USERS))
      .then((response) => {
        const { users } = response;

        return users.map((user) => {
          return helpers.addGravatarUrlToResource(user);
        });
      });
  }

  inviteUser = (formData) => {
    const { INVITES } = endpoints;

    return this.authenticatedPost(this.endpoint(INVITES), JSON.stringify(formData))
      .then((response) => {
        const { invite } = response;

        return helpers.addGravatarUrlToResource(invite);
      });
  }

  loginUser ({ username, password }) {
    const { LOGIN } = endpoints;
    const loginEndpoint = this.baseURL + LOGIN;

    return Base.post(loginEndpoint, JSON.stringify({ username, password }))
      .then((response) => {
        const { user } = response;
        const userWithGravatarUrl = helpers.addGravatarUrlToResource(user);

        return {
          ...response,
          user: userWithGravatarUrl,
        };
      });
  }

  logout () {
    const { LOGOUT } = endpoints;
    const logoutEndpoint = this.baseURL + LOGOUT;

    return this.authenticatedPost(logoutEndpoint);
  }

  me () {
    const { ME } = endpoints;
    const meEndpoint = this.baseURL + ME;

    return this.authenticatedGet(meEndpoint)
      .then((response) => {
        const { user } = response;

        return helpers.addGravatarUrlToResource(user);
      });
  }

  resetPassword (formData) {
    const { RESET_PASSWORD } = endpoints;
    const resetPasswordEndpoint = this.baseURL + RESET_PASSWORD;

    return Base.post(resetPasswordEndpoint, JSON.stringify(formData));
  }

  revokeInvite = ({ id }) => {
    const { INVITES } = endpoints;
    const endpoint = `${this.endpoint(INVITES)}/${id}`;

    return this.authenticatedDelete(endpoint);
  }

  runQuery = ({ query, selected }) => {
    const { RUN_QUERY } = endpoints;

    return this.authenticatedPost(this.endpoint(RUN_QUERY), JSON.stringify({ query, selected }))
      .then(response => response.campaign);
  }

  runQueryWebsocket = (campaignID) => {
    return new Promise((resolve) => {
      const socket = new global.WebSocket(`${this.websocketBaseURL}/v1/kolide/results/${campaignID}`);

      socket.onopen = () => {
        socket.send(JSON.stringify({ type: 'auth', data: { token: local.getItem('auth_token') } }));
      };

      return resolve(socket);
    });
  }

  setup = (formData) => {
    const { SETUP } = endpoints;
    const setupData = helpers.setupData(formData);

    return Base.post(this.endpoint(SETUP), JSON.stringify(setupData));
  }

  updateConfig = (formData) => {
    const { CONFIG } = endpoints;
    const configData = helpers.formatConfigDataForServer(formData);

    if (get(configData, 'smtp_settings.port')) {
      configData.smtp_settings.port = parseInt(configData.smtp_settings.port, 10);
    }

    return this.authenticatedPatch(this.endpoint(CONFIG), JSON.stringify(configData));
  }

  updatePack = (pack, { description, name, targets }) => {
    const { PACKS } = endpoints;
    const updatePackEndpoint = `${this.baseURL}${PACKS}/${pack.id}`;
    const packTargets = helpers.formatSelectedTargetsForApi(targets, true);

    return this.authenticatedPatch(updatePackEndpoint, JSON.stringify({ description, name, ...packTargets }))
      .then((response) => { return response.pack; });
  }

  updateQuery = ({ id: queryID }, updateParams) => {
    const { QUERIES } = endpoints;
    const updateQueryEndpoint = `${this.baseURL}${QUERIES}/${queryID}`;

    return this.authenticatedPatch(updateQueryEndpoint, JSON.stringify(updateParams))
      .then((response) => { return response.query; });
  }

  updateUser = (user, formData) => {
    const { USERS } = endpoints;
    const updateUserEndpoint = `${this.baseURL}${USERS}/${user.id}`;

    return this.authenticatedPatch(updateUserEndpoint, JSON.stringify(formData))
      .then((response) => {
        const { user: updatedUser } = response;

        return helpers.addGravatarUrlToResource(updatedUser);
      });
  }

  requirePasswordReset = (user, { require }) => {
    const { USERS } = endpoints;
    const requirePasswordResetEndpoint = this.endpoint(`${USERS}/${user.id}/require_password_reset`);

    return this.authenticatedPost(requirePasswordResetEndpoint, JSON.stringify({ require }))
      .then((response) => {
        const { user: updatedUser } = response;
        return helpers.addGravatarUrlToResource(updatedUser);
      });
  }

  // Perform a password reset for the currently logged in user that has had a
  // reset required
  performRequiredPasswordReset = ({ password }) => {
    const { PERFORM_REQUIRED_PASSWORD_RESET } = endpoints;
    const performRequiredPasswordResetEndpoint = this.baseURL + PERFORM_REQUIRED_PASSWORD_RESET;

    return this.authenticatedPost(performRequiredPasswordResetEndpoint, JSON.stringify({ new_password: password }))
      .then((response) => {
        const { user: updatedUser } = response;
        return helpers.addGravatarUrlToResource(updatedUser);
      });
  }
}

export default new Kolide();
