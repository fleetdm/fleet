import nock from 'nock';

import helpers from 'kolide/helpers';
import Kolide from 'kolide';
import mocks from 'test/mocks';

const { config: configMocks } = mocks;

describe('Kolide - API client (config)', () => {
  afterEach(() => {
    nock.cleanAll();
    Kolide.setBearerToken(null);
  });

  const bearerToken = 'valid-bearer-token';

  describe('#loadAll', () => {
    it('calls the appropriate endpoint with the correct parameters', () => {
      const request = configMocks.loadAll.valid(bearerToken);

      Kolide.setBearerToken(bearerToken);
      return Kolide.config.loadAll()
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });

  describe('#update', () => {
    it('calls the appropriate endpoint with the correct parameters', () => {
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
      };
      const configData = helpers.formatConfigDataForServer(formData);
      const request = configMocks.update.valid(bearerToken, configData);

      Kolide.setBearerToken(bearerToken);
      return Kolide.config.update(formData)
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });
});
