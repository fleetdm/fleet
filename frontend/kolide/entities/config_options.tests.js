import expect from 'expect';
import nock from 'nock';

import { configOptionStub } from 'test/stubs';
import Kolide from 'kolide';
import mocks from 'test/mocks';

const { configOptions: configOptionMocks } = mocks;

describe('Kolide - API client (config options)', () => {
  afterEach(() => {
    nock.cleanAll();
    Kolide.setBearerToken(null);
  });

  const bearerToken = 'valid-bearer-token';

  describe('#loadAll', () => {
    it('calls the appropriate endpoint with the correct parameters', () => {
      const request = configOptionMocks.loadAll.valid(bearerToken);

      Kolide.setBearerToken(bearerToken);
      return Kolide.configOptions.loadAll()
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });

  describe('#update', () => {
    it('calls the appropriate endpoint with the correct parameters', () => {
      const options = [configOptionStub];
      const request = configOptionMocks.update.valid(bearerToken, options);

      Kolide.setBearerToken(bearerToken);
      return Kolide.configOptions.update(options)
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });
});
