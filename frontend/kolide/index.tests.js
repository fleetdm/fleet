import expect from 'expect';
import nock from 'nock';

import Kolide from 'kolide';
import helpers from 'kolide/helpers';
import mocks from 'test/mocks';
import { configOptionStub, hostStub, packStub, queryStub, userStub, labelStub } from 'test/stubs';

const {
  invalidForgotPasswordRequest,
  invalidResetPasswordRequest,
  validChangePasswordRequest,
  validCreateLabelRequest,
  validCreatePackRequest,
  validCreateQueryRequest,
  validCreateScheduledQueryRequest,
  validDestroyHostRequest,
  validDestroyLabelRequest,
  validDestroyQueryRequest,
  validDestroyPackRequest,
  validDestroyScheduledQueryRequest,
  validEnableUserRequest,
  validForgotPasswordRequest,
  validGetConfigOptionsRequest,
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
  validStatusLabelsGetCountsRequest,
  validUpdateAdminRequest,
  validUpdateConfigOptionsRequest,
  validUpdateConfigRequest,
  validUpdatePackRequest,
  validUpdateQueryRequest,
  validUpdateUserRequest,
  validUser,
} = mocks;

describe('Kolide - API client', () => {
  afterEach(() => {
    nock.cleanAll();
    Kolide.setBearerToken(null);
  });

  const bearerToken = 'valid-bearer-token';

  describe('defaults', () => {
    it('sets the base URL', () => {
      expect(Kolide.baseURL).toEqual('http://localhost:8080/api');
    });
  });

  describe('statusLabels', () => {
    it('#getCounts', (done) => {
      const request = validStatusLabelsGetCountsRequest(bearerToken);

      Kolide.setBearerToken(bearerToken);
      Kolide.statusLabels.getCounts()
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(() => {
          throw new Error('Endpoint not reached');
        });
    });
  });

  describe('labels', () => {
    describe('#create', () => {
      it('calls the appropriate endpoint with the correct parameters', (done) => {
        const description = 'label description';
        const name = 'label name';
        const query = 'SELECT * FROM users';
        const labelParams = { description, name, query };
        const request = validCreateLabelRequest(bearerToken, labelParams);

        Kolide.setBearerToken(bearerToken);
        Kolide.labels.create(labelParams)
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

    describe('#destroy', () => {
      it('calls the appropriate endpoint with the correct parameters', (done) => {
        const request = validDestroyLabelRequest(bearerToken, labelStub);

        Kolide.setBearerToken(bearerToken);
        Kolide.labels.destroy(labelStub)
          .then(() => {
            expect(request.isDone()).toEqual(true);
            done();
          })
          .catch(() => {
            throw new Error('Request should have been stubbed');
          });
      });
    });
  });

  describe('configOptions', () => {
    it('#loadAll', (done) => {
      const request = validGetConfigOptionsRequest(bearerToken);

      Kolide.setBearerToken(bearerToken);
      Kolide.configOptions.loadAll()
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(() => {
          throw new Error('Request should have been stubbed');
        });
    });

    it('#update', (done) => {
      const options = [configOptionStub];
      const request = validUpdateConfigOptionsRequest(bearerToken, options);

      Kolide.setBearerToken(bearerToken);
      Kolide.configOptions.update(options)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(() => {
          throw new Error('Request should have been stubbed');
        });
    });
  });

  describe('hosts', () => {
    describe('#loadAll', () => {
      it('calls the correct endpoint with the correct params', (done) => {
        const request = validGetHostsRequest(bearerToken);

        Kolide.setBearerToken(bearerToken);
        Kolide.hosts.loadAll()
          .then(() => {
            expect(request.isDone()).toEqual(true);
            done();
          })
          .catch(done);
      });
    });

    describe('#destroy', () => {
      it('calls the correct endpoint with the correct params', (done) => {
        const request = validDestroyHostRequest(bearerToken, hostStub);

        Kolide.setBearerToken(bearerToken);
        Kolide.hosts.destroy(hostStub)
          .then(() => {
            expect(request.isDone()).toEqual(true);
            done();
          })
          .catch(() => {
            throw new Error('Expected the request to be stubbed');
          });
      });
    });
  });

  describe('packs', () => {
    it('#createPack', (done) => {
      const { description, name } = packStub;
      const params = { description, name, host_ids: [], label_ids: [] };
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
      const request = validDestroyPackRequest(bearerToken, packStub);

      Kolide.setBearerToken(bearerToken);
      Kolide.destroyPack(packStub)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    describe('#updatePack', () => {
      it('sends the host and/or label ids if packs are changed', (done) => {
        const targets = [hostStub];
        const updatePackParams = { name: 'New Pack Name', host_ids: [hostStub.id] };
        const request = validUpdatePackRequest(bearerToken, packStub, updatePackParams);
        const updatedPack = { name: 'New Pack Name', targets };

        Kolide.setBearerToken(bearerToken);
        Kolide.updatePack(packStub, updatedPack)
          .then(() => {
            expect(request.isDone()).toEqual(true);
            done();
          })
          .catch(done);
      });

      it('does not send the host or label ids if packs are not changed', (done) => {
        const updatePackParams = { name: 'New Pack Name' };
        const request = validUpdatePackRequest(bearerToken, packStub, updatePackParams);

        Kolide.setBearerToken(bearerToken);
        Kolide.updatePack(packStub, updatePackParams)
          .then(() => {
            expect(request.isDone()).toEqual(true);
            done();
          })
          .catch(done);
      });
    });
  });

  describe('queries', () => {
    it('#createQuery', (done) => {
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

    it('#destroyQuery', (done) => {
      const request = validDestroyQueryRequest(bearerToken, queryStub);

      Kolide.setBearerToken(bearerToken);
      Kolide.destroyQuery(queryStub)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it('getQueries', (done) => {
      const request = validGetQueriesRequest(bearerToken);

      Kolide.setBearerToken(bearerToken);
      Kolide.getQueries()
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it('#getQuery', (done) => {
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

    describe('#run', () => {
      it('calls the correct endpoint with the correct params', (done) => {
        const data = { query: 'select * from users', selected: { hosts: [], labels: [] } };
        const request = validRunQueryRequest(bearerToken, data);

        Kolide.setBearerToken(bearerToken);
        Kolide.queries.run(data)
          .then(() => {
            expect(request.isDone()).toEqual(true);
            done();
          })
          .catch(done);
      });
    });
  });

  describe('users', () => {
    describe('#changePassword', () => {
      it('calls the appropriate endpoint with the correct parameters', (done) => {
        const passwordParams = { old_password: 'password', new_password: 'p@ssw0rd' };
        const request = validChangePasswordRequest(bearerToken, passwordParams);

        Kolide.setBearerToken(bearerToken);
        Kolide.users.changePassword(passwordParams)
          .then(() => {
            expect(request.isDone()).toEqual(true);
            done();
          })
          .catch(() => {
            throw new Error('Expected request to have been stubbed');
          });
      });
    });

    describe('#enable', () => {
      it('calls the appropriate endpoint with the correct parameters', (done) => {
        const enableParams = { enabled: true };
        const request = validEnableUserRequest(bearerToken, userStub, enableParams);

        Kolide.setBearerToken(bearerToken);
        Kolide.users.enable(userStub, enableParams)
          .then(() => {
            expect(request.isDone()).toEqual(true);
            done();
          })
          .catch(() => {
            throw new Error('Request should have been stubbed');
          });
      });
    });

    describe('#updateAdmin', () => {
      it('calls the appropriate endpoint with the correct parameters', (done) => {
        const adminParams = { admin: true };
        const request = validUpdateAdminRequest(bearerToken, userStub, adminParams);

        Kolide.setBearerToken(bearerToken);
        Kolide.users.updateAdmin(userStub, adminParams)
          .then(() => {
            expect(request.isDone()).toEqual(true);
            done();
          })
          .catch(() => {
            throw new Error('Request should have been stubbed');
          });
      });
    });
  });

  describe('#getConfig', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const request = validGetConfigRequest(bearerToken);

      Kolide.getConfig(bearerToken)
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#getInvites', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
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

  describe('#createScheduledQuery', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const formData = {
        interval: 60,
        logging_type: 'differential',
        pack_id: 1,
        platform: 'darwin',
        query_id: 2,
        shard: 12,
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
      const request = validRevokeInviteRequest(bearerToken, userStub);

      Kolide.setBearerToken(bearerToken);
      Kolide.revokeInvite(userStub)
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
