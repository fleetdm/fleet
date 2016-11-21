import Base from './base';
import endpoints from './endpoints';
import { appendTargetTypeToTargets } from '../redux/nodes/entities/targets/helpers';

class Kolide extends Base {
  createLabel = ({ description, name, query }) => {
    const { LABELS } = endpoints;

    return this.authenticatedPost(this.endpoint(LABELS), JSON.stringify({ description, name, query }))
      .then((response) => {
        return {
          ...response.label,
          type: 'custom',
        };
      });
  }

  createQuery = ({ description, name, query }) => {
    const { QUERIES } = endpoints;

    return this.authenticatedPost(this.endpoint(QUERIES), JSON.stringify({ description, name, query }))
      .then((response) => { return response.query; });
  }

  forgotPassword ({ email }) {
    const { FORGOT_PASSWORD } = endpoints;
    const forgotPasswordEndpoint = this.baseURL + FORGOT_PASSWORD;

    return Base.post(forgotPasswordEndpoint, JSON.stringify({ email }));
  }

  getConfig = () => {
    const { CONFIG } = endpoints;

    return this.authenticatedGet(this.endpoint(CONFIG))
      .then((response) => { return response.org_info; });
  }

  getInvites = () => {
    const { INVITES } = endpoints;

    return this.authenticatedGet(this.endpoint(INVITES))
      .then((response) => { return response.invites; });
  }

  getHosts = () => {
    const { HOSTS } = endpoints;

    return this.authenticatedGet(this.endpoint(HOSTS))
      .then((response) => { return response.hosts; });
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

  getQuery = (queryID) => {
    const { QUERIES } = endpoints;
    const getQueryEndpoint = `${this.baseURL}${QUERIES}/${queryID}`;

    return this.authenticatedGet(getQueryEndpoint)
      .then((response) => { return response.query; });
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
        const labels = response.labels.map((label) => {
          return { ...label, type: 'custom' };
        });
        const stubbedLabels = [
          { id: 100, display_text: 'All Hosts', type: 'all', count: 22 },
          { id: 40, display_text: 'OFFLINE', type: 'status', count: 2 },
          { id: 50, display_text: 'ONLINE', type: 'status', count: 20 },
          { id: 60, display_text: 'MAC OS', type: 'platform', count: 1 },
          { id: 70, display_text: 'CENTOS', type: 'platform', count: 10 },
          { id: 80, display_text: 'UBUNTU', type: 'platform', count: 10 },
          { id: 110, display_text: 'WINDOWS', type: 'platform', count: 1 },
        ];

        return labels.concat(stubbedLabels);
      });
  }

  getPacks = () => {
    const { PACKS } = endpoints;

    return this.authenticatedGet(this.endpoint(PACKS))
      .then((response) => { return response.packs; });
  }

  getUsers = () => {
    const { USERS } = endpoints;

    return this.authenticatedGet(this.endpoint(USERS))
      .then((response) => { return response.users; });
  }

  inviteUser = (formData) => {
    const { INVITES } = endpoints;

    return this.authenticatedPost(this.endpoint(INVITES), JSON.stringify(formData))
      .then((response) => { return response.invite; });
  }

  loginUser ({ username, password }) {
    const { LOGIN } = endpoints;
    const loginEndpoint = this.baseURL + LOGIN;

    return Base.post(loginEndpoint, JSON.stringify({ username, password }));
  }

  logout () {
    const { LOGOUT } = endpoints;
    const logoutEndpoint = this.baseURL + LOGOUT;

    return this.authenticatedPost(logoutEndpoint);
  }

  me () {
    const { ME } = endpoints;
    const meEndpoint = this.baseURL + ME;

    return this.authenticatedGet(meEndpoint);
  }

  resetPassword (formData) {
    const { RESET_PASSWORD } = endpoints;
    const resetPasswordEndpoint = this.baseURL + RESET_PASSWORD;

    return Base.post(resetPasswordEndpoint, JSON.stringify(formData));
  }

  revokeInvite = ({ entityID }) => {
    const { INVITES } = endpoints;
    const endpoint = `${this.endpoint(INVITES)}/${entityID}`;

    return this.authenticatedDelete(endpoint);
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
      .then((response) => { return response.user; });
  }
}

export default new Kolide();
