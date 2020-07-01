import expect from 'expect';
import nock from 'nock';

import Kolide from 'kolide';
import mocks from 'test/mocks';
import createRequestMock from 'test/mocks/create_request_mock';
import { hostStub } from 'test/stubs';

const { hosts: hostMocks } = mocks;

describe('Kolide - API client (hosts)', () => {
  afterEach(() => {
    nock.cleanAll();
    Kolide.setBearerToken(null);
  });

  const bearerToken = 'valid-bearer-token';

  describe('#destroy', () => {
    it('calls the correct endpoint with the correct params', () => {
      const request = hostMocks.destroy.valid(bearerToken, hostStub);

      Kolide.setBearerToken(bearerToken);
      return Kolide.hosts.destroy(hostStub)
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });

  describe('#loadAll', () => {
    it('calls the correct endpoint with the correct params', () => {
      const request = hostMocks.loadAll.valid(bearerToken);

      Kolide.setBearerToken(bearerToken);
      return Kolide.hosts.loadAll()
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });

    it('calls the label endpoint when used with label filter', () => {
      const request = createRequestMock({
        bearerToken,
        endpoint: '/api/v1/kolide/labels/6/hosts?page=1&per_page=50&order_key=host_name',
        method: 'get',
        response: { hosts: [] },
      });

      Kolide.setBearerToken(bearerToken);
      return Kolide.hosts.loadAll(2, 50, 'labels/6')
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });
});
