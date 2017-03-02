import createRequestMock from 'test/mocks/create_request_mock';
import { licenseStub } from 'test/stubs';

export default {
  create: {
    valid: (bearerToken, jwtToken, response = licenseStub()) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/license',
        params: { license: jwtToken },
        method: 'post',
        response: { license: { ...response, token: jwtToken } },
        responseStatus: 201,
      });
    },
  },
  load: {
    valid: (bearerToken, license = licenseStub()) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/license',
        method: 'get',
        response: { license },
      });
    },
  },
  setup: {
    valid: (bearerToken, jwtToken, response = licenseStub()) => {
      return createRequestMock({
        bearerToken,
        endpoint: '/api/v1/license',
        method: 'post',
        params: { license: jwtToken },
        response: { license: { ...response, token: jwtToken } },
        responseStatus: 201,
      });
    },
  },
};
