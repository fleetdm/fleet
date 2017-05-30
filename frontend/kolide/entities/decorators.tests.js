import expect from 'expect';
import nock from 'nock';

import Kolide from 'kolide';
import decoratorsMocks from 'test/mocks/decorators_mocks';

describe('Kolide - api client (decorators)', () => {
  afterEach(() => {
    nock.cleanAll();
    Kolide.setBearerToken(null);
  });

  const bearerToken = 'valid-bearer-token';

  describe('#loadAll', () => {
    it('calls the appropriate endpoint with the correct parameters', () => {
      const request = decoratorsMocks.loadAll.valid(bearerToken);

      Kolide.setBearerToken(bearerToken);
      return Kolide.decorators.loadAll()
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });

  describe('#create', () => {
    it('calls the appropriate endpoint with the correct parameters', () => {
      const query = 'SELECT FROM FOO;';
      const interval = 0;
      const param = { name: 'foo', type: 'load', query, interval, built_in: false };
      const request = decoratorsMocks.create.valid(bearerToken, param);
      Kolide.setBearerToken(bearerToken);
      return Kolide.decorators.create(param)
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });

  describe('#destroy', () => {
    it('calls the appropriate endpoint with the correct parameters', () => {
      const id = 1;
      const param = { id };
      const request = decoratorsMocks.destroy.valid(bearerToken, param);
      Kolide.setBearerToken(bearerToken);
      return Kolide.decorators.destroy(param)
        .then(() => {
          expect(request.isDone()).toEqual(true);
        });
    });
  });
});
