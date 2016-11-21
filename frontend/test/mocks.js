import nock from 'nock';

import helpers from 'kolide/helpers';

export const validUser = {
  token: 'auth_token',
  id: 1,
  username: 'admin',
  email: 'admin@kolide.co',
  name: '',
  admin: true,
  enabled: true,
  needs_password_reset: false,
  gravatarURL: 'https://www.gravatar.com/avatar/7157f4758f8423b59aaee869d919f6b9',
};

export const validCreateLabelRequest = (bearerToken, labelParams) => {
  return nock('http://localhost:8080', {
    reqHeaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
    .post('/api/v1/kolide/labels', JSON.stringify(labelParams))
    .reply(201, { label: labelParams });
};

export const validCreateQueryRequest = (bearerToken, queryParams) => {
  return nock('http://localhost:8080', {
    reqHeaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
    .post('/api/v1/kolide/queries', JSON.stringify(queryParams))
    .reply(201, { query: queryParams });
};

export const invalidGetQueryRequest = (bearerToken, queryID) => {
  return nock('http://localhost:8080', {
    reqHeaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
    .get(`/api/v1/kolide/queries/${queryID}`)
    .reply(404, { error: 'resource not found' });
};

export const validGetQueryRequest = (bearerToken, queryID) => {
  return nock('http://localhost:8080', {
    reqHeaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
    .get(`/api/v1/kolide/queries/${queryID}`)
    .reply(200, { query: { id: queryID } });
};

export const validGetConfigRequest = (bearerToken) => {
  return nock('http://localhost:8080', {
    reqHeaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
    .get('/api/v1/kolide/config')
    .reply(200, { config: { name: 'Kolide' } });
};

export const validGetInvitesRequest = (bearerToken) => {
  return nock('http://localhost:8080', {
    reqHeaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
    .get('/api/v1/kolide/invites')
    .reply(200, {
      invites: [
        { name: 'Joe Schmoe', email: 'joe@schmoe.org', admin: false },
      ],
    });
};

export const validInviteUserRequest = (bearerToken, formData) => {
  return nock('http://localhost:8080', {
    reqHeaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
    .post('/api/v1/kolide/invites', JSON.stringify(formData))
    .reply(200, formData);
};

export const validGetHostsRequest = (bearerToken) => {
  return nock('http://localhost:8080', {
    reqHeaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
    .get('/api/v1/kolide/hosts')
    .reply(200, { hosts: [] });
};

export const validGetTargetsRequest = (bearerToken, query) => {
  return nock('http://localhost:8080', {
    reqHeaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
    .post('/api/v1/kolide/targets', {
      query,
      selected: {
        hosts: [],
        labels: [],
      },
    })
    .reply(200, {
      selected_targets_count: 1234,
      targets: [
        {
          id: 3,
          label: 'OS X El Capitan 10.11',
          name: 'osx-10.11',
          platform: 'darwin',
          target_type: 'hosts',
        },
        {
          id: 4,
          label: 'Jason Meller\'s Macbook Pro',
          name: 'jmeller.local',
          platform: 'darwin',
          target_type: 'hosts',
        },
        {
          id: 4,
          label: 'All Macs',
          name: 'macs',
          count: 1234,
          target_type: 'labels',
        },
      ],
    });
};

export const validGetUsersRequest = (bearerToken) => {
  return nock('http://localhost:8080', {
    reqHeaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
    .get('/api/v1/kolide/users')
    .reply(200, { users: [validUser] });
};

export const validLoginRequest = () => {
  return nock('http://localhost:8080')
  .post('/api/v1/kolide/login')
  .reply(200, validUser);
};

export const validMeRequest = (bearerToken) => {
  return nock('http://localhost:8080', {
    reqheaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
    .get('/api/v1/kolide/me')
    .reply(200, validUser);
};

export const validLogoutRequest = (bearerToken) => {
  return nock('http://localhost:8080', {
    reqheaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
    .post('/api/v1/kolide/logout')
    .reply(200, {});
};

export const validForgotPasswordRequest = () => {
  return nock('http://localhost:8080')
  .post('/api/v1/kolide/forgot_password')
  .reply(200, validUser);
};

export const invalidForgotPasswordRequest = (error) => {
  return nock('http://localhost:8080')
  .post('/api/v1/kolide/forgot_password')
  .reply(422, { error });
};

export const validResetPasswordRequest = (password, token) => {
  return nock('http://localhost:8080')
  .post('/api/v1/kolide/reset_password', JSON.stringify({
    new_password: password,
    password_reset_token: token,
  }))
  .reply(200, validUser);
};

export const validRevokeInviteRequest = (bearerToken, inviteID) => {
  return nock('http://localhost:8080', {
    reqheaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
  .delete(`/api/v1/kolide/invites/${inviteID}`)
  .reply(200);
};

export const invalidResetPasswordRequest = (password, token, error) => {
  return nock('http://localhost:8080')
  .post('/api/v1/kolide/reset_password', JSON.stringify({
    new_password: password,
    password_reset_token: token,
  }))
  .reply(422, { error });
};

export const validSetupRequest = (formData) => {
  const setupData = helpers.setupData(formData);

  return nock('http://localhost:8080')
    .post('/api/v1/setup', JSON.stringify(setupData))
    .reply(200, {});
};

export const validUpdateQueryRequest = (bearerToken, query, formData) => {
  return nock('http://localhost:8080', {
    reqHeaders: {
      Authorization: `Bearer ${bearerToken}`,
    },
  })
  .patch(`/api/v1/kolide/queries/${query.id}`, JSON.stringify(formData))
  .reply(200, { query: { ...query, ...formData } });
};

export const validUpdateUserRequest = (user, formData) => {
  return nock('http://localhost:8080')
  .patch(`/api/v1/kolide/users/${user.id}`, JSON.stringify(formData))
  .reply(200, validUser);
};


export default {
  invalidForgotPasswordRequest,
  invalidGetQueryRequest,
  invalidResetPasswordRequest,
  validCreateLabelRequest,
  validCreateQueryRequest,
  validForgotPasswordRequest,
  validGetConfigRequest,
  validGetHostsRequest,
  validGetInvitesRequest,
  validGetQueryRequest,
  validGetTargetsRequest,
  validGetUsersRequest,
  validInviteUserRequest,
  validLoginRequest,
  validLogoutRequest,
  validMeRequest,
  validResetPasswordRequest,
  validRevokeInviteRequest,
  validSetupRequest,
  validUpdateQueryRequest,
  validUpdateUserRequest,
  validUser,
};
