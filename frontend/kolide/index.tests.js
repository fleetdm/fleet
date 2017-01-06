import expect from 'expect';
import nock from 'nock';

import Kolide from 'kolide';
import helpers from 'kolide/helpers';
import mocks from 'test/mocks';

const {
  invalidForgotPasswordRequest,
  invalidResetPasswordRequest,
  validCreateLabelRequest,
  validCreatePackRequest,
  validCreateQueryRequest,
  validCreateScheduledQueryRequest,
  validDestroyPackRequest,
  validDestroyScheduledQueryRequest,
  validForgotPasswordRequest,
  validGetConfigRequest,
  validGetHostsRequest,
  validGetInvitesRequest,
  validGetQueriesRequest,
  validGetQueryRequest,
  validGetScheduledQueriesRequest,
  validGetTargetsRequest,
  validGetUsersRequest,
  validInviteUserRequest,
  validLoginRequest,
  validLogoutRequest,
  validMeRequest,
  validResetPasswordRequest,
  validRevokeInviteRequest,
  validRunQueryRequest,
  validSetupRequest,
  validUpdateConfigRequest,
  validUpdatePackRequest,
  validUpdateQueryRequest,
  validUpdateUserRequest,
  validUser,
} = mocks;

describe('Kolide - API client', () => {
  afterEach(() => { nock.cleanAll(); });

  describe('defaults', () => {
    it('sets the base URL', () => {
      expect(Kolide.baseURL).toEqual('http://localhost:8080/api');
    });
  });

  describe('#createLabel', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const description = 'label description';
      const name = 'label name';
      const query = 'SELECT * FROM users';
      const labelParams = { description, name, query };
      const request = validCreateLabelRequest(bearerToken, labelParams);

      Kolide.setBearerToken(bearerToken);
      Kolide.createLabel(labelParams)
        .then((labelResponse) => {
          expect(request.isDone()).toEqual(true);
          expect(labelResponse).toEqual({
            ...labelParams,
            display_text: name,
            slug: 'label-name',
            type: 'custom',
          });
          done();
        })
        .catch(done);
    });
  });

  describe('packs', () => {
    const bearerToken = 'valid-bearer-token';
    const pack = { id: 1, name: 'Pack Name', description: 'Pack Description' };

    it('#createPack', (done) => {
      const { description, name } = pack;
      const params = { description, name };
      const request = validCreatePackRequest(bearerToken, params);

      Kolide.setBearerToken(bearerToken);
      Kolide.createPack(params)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it('#destroyPack', (done) => {
      const request = validDestroyPackRequest(bearerToken, pack);

      Kolide.setBearerToken(bearerToken);
      Kolide.destroyPack(pack)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it('#updatePack', (done) => {
      const updatePackParams = { name: 'New Pack Name' };
      const request = validUpdatePackRequest(bearerToken, pack, updatePackParams);

      Kolide.setBearerToken(bearerToken);
      Kolide.updatePack(pack, updatePackParams)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#createQuery', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const description = 'query description';
      const name = 'query name';
      const query = 'SELECT * FROM users';
      const queryParams = { description, name, query };
      const request = validCreateQueryRequest(bearerToken, queryParams);

      Kolide.setBearerToken(bearerToken);
      Kolide.createQuery(queryParams)
        .then((queryResponse) => {
          expect(request.isDone()).toEqual(true);
          expect(queryResponse).toEqual(queryParams);
          done();
        })
        .catch(done);
    });
  });

  describe('#getConfig', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const request = validGetConfigRequest(bearerToken);

      Kolide.getConfig(bearerToken)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#getHosts', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const request = validGetHostsRequest(bearerToken);

      Kolide.setBearerToken(bearerToken);
      Kolide.getHosts()
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#getInvites', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const request = validGetInvitesRequest(bearerToken);

      Kolide.setBearerToken(bearerToken);
      Kolide.getInvites()
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#getQueries', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const request = validGetQueriesRequest(bearerToken);

      Kolide.setBearerToken(bearerToken);
      Kolide.getQueries()
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#getQuery', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const queryID = 10;
      const request = validGetQueryRequest(bearerToken, queryID);

      Kolide.setBearerToken(bearerToken);
      Kolide.getQuery(queryID)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#createScheduledQuery', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const formData = {
        interval: 60,
        logging_type: 'differential',
        pack_id: 1,
        platform: 'darwin',
        query_id: 2,
      };
      const request = validCreateScheduledQueryRequest(bearerToken, formData);

      Kolide.setBearerToken(bearerToken);
      Kolide.createScheduledQuery(formData)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#destroyScheduledQuery', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const scheduledQuery = { id: 1 };
      const request = validDestroyScheduledQueryRequest(bearerToken, scheduledQuery);

      Kolide.setBearerToken(bearerToken);
      Kolide.destroyScheduledQuery(scheduledQuery)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#getScheduledQueries', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const pack = { id: 1 };
      const request = validGetScheduledQueriesRequest(bearerToken, pack);

      Kolide.setBearerToken(bearerToken);
      Kolide.getScheduledQueries(pack)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#getTargets', () => {
    it('correctly parses the response', (done) => {
      const bearerToken = 'valid-bearer-token';
      const hosts = [];
      const labels = [];
      const query = 'mac';
      const request = validGetTargetsRequest(bearerToken, query);

      Kolide.setBearerToken(bearerToken);
      Kolide.getTargets(query, { hosts, labels })
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#getUsers', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const request = validGetUsersRequest();

      Kolide.getUsers(bearerToken)
        .then((users) => {
          expect(users).toEqual([validUser]);
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#inviteUser', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const formData = {
        email: 'new@user.org',
        admin: false,
        invited_by: 1,
        id: 1,
        name: '',
      };
      const request = validInviteUserRequest(bearerToken, formData);

      Kolide.setBearerToken(bearerToken);
      Kolide.inviteUser(formData)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#me', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'ABC123';
      const request = validMeRequest(bearerToken);

      Kolide.setBearerToken(bearerToken);
      Kolide.me()
        .then((user) => {
          expect(user).toEqual(validUser);
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#loginUser', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const request = validLoginRequest();

      Kolide.loginUser({
        username: 'admin',
        password: 'secret',
      })
        .then(({ user }) => {
          expect(user).toEqual(validUser);
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#logout', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'ABC123';
      const request = validLogoutRequest(bearerToken);

      Kolide.setBearerToken(bearerToken);
      Kolide.logout()
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#forgotPassword', () => {
    it('calls the appropriate endpoint with the correct parameters when successful', (done) => {
      const request = validForgotPasswordRequest();
      const email = 'hi@thegnar.co';

      Kolide.forgotPassword({ email })
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it('return errors correctly for unsuccessful requests', (done) => {
      const error = { base: 'Something went wrong' };
      const errorResponse = {
        message: {
          errors: [error],
        },
      };
      const request = invalidForgotPasswordRequest(errorResponse);
      const email = 'hi@thegnar.co';

      Kolide.forgotPassword({ email })
        .then(done)
        .catch(() => {
          expect(request.isDone()).toEqual(true);
          done();
        });
    });
  });

  describe('#resetPassword', () => {
    const newPassword = 'p@ssw0rd';

    it('calls the appropriate endpoint with the correct parameters when successful', (done) => {
      const passwordResetToken = 'password-reset-token';
      const request = validResetPasswordRequest(newPassword, passwordResetToken);
      const formData = {
        new_password: newPassword,
        password_reset_token: passwordResetToken,
      };

      Kolide.resetPassword(formData)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it('return errors correctly for unsuccessful requests', (done) => {
      const error = 'Resource not found';
      const passwordResetToken = 'invalid-password-reset-token';
      const request = invalidResetPasswordRequest(newPassword, passwordResetToken, error);
      const formData = {
        new_password: newPassword,
        password_reset_token: passwordResetToken,
      };

      Kolide.resetPassword(formData)
        .then(done)
        .catch((errorResponse) => {
          const { response } = errorResponse;

          expect(response).toEqual({ error });
          expect(request.isDone()).toEqual(true);
          done();
        });
    });
  });

  describe('#revokeInvite', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const entityID = 1;
      const request = validRevokeInviteRequest(bearerToken, entityID);

      Kolide.setBearerToken(bearerToken);
      Kolide.revokeInvite({ entityID })
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#runQuery', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const data = { query: 'select * from users', selected: { hosts: [], labels: [] } };
      const request = validRunQueryRequest(bearerToken, data);

      Kolide.setBearerToken(bearerToken);
      Kolide.runQuery(data)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#setup', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const formData = {
        email: 'hi@gnar.dog',
        name: 'Gnar Dog',
        kolide_server_url: 'https://gnar.kolide.co',
        org_logo_url: 'https://thegnar.co/assets/logo.png',
        org_name: 'The Gnar Co.',
        password: 'p@ssw0rd',
        password_confirmation: 'p@ssw0rd',
        username: 'gnardog',
      };
      const request = validSetupRequest(formData);

      Kolide.setup(formData)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#updateConfig', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const formData = {
        org_name: 'Kolide',
        org_logo_url: '0.0.0.0:8080/logo.png',
        kolide_server_url: '',
        configured: false,
        sender_address: '',
        server: '',
        port: 587,
        authentication_type: 'authtype_username_password',
        user_name: '',
        password: '',
        enable_ssl_tls: true,
        authentication_method: 'authmethod_plain',
        verify_ssl_certs: true,
        enable_start_tls: true,
        email_enabled: false,
      };
      const configData = helpers.formatConfigDataForServer(formData);
      const request = validUpdateConfigRequest(bearerToken, configData);

      Kolide.setBearerToken(bearerToken);
      Kolide.updateConfig(formData)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#updateQuery', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const bearerToken = 'valid-bearer-token';
      const query = { id: 1, name: 'Query Name', description: 'Query Description', query: 'SELECT * FROM users' };
      const updateQueryParams = { name: 'New Query Name' };
      const request = validUpdateQueryRequest(bearerToken, query, updateQueryParams);

      Kolide.setBearerToken(bearerToken);
      Kolide.updateQuery(query, updateQueryParams)
        .then((queryResponse) => {
          expect(request.isDone()).toEqual(true);
          expect(queryResponse).toEqual({
            ...query,
            ...updateQueryParams,
          });
          done();
        })
        .catch(done);
    });
  });

  describe('#updateUser', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const formData = { enabled: false };
      const request = validUpdateUserRequest(validUser, formData);

      Kolide.updateUser(validUser, formData)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });
});
