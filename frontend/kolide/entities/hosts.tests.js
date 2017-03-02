import expect from 'expect';
import nock from 'nock';

import Kolide from 'kolide';
import mocks from 'test/mocks';
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
  });
});
