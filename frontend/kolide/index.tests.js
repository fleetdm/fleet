import expect from 'expect';
import Kolide from './index';
import { validLoginRequest } from '../test/mocks';

describe('Kolide - API client', () => {
  describe('defaults', () => {
    it('sets the base URL', () => {
      expect(Kolide.baseURL).toEqual('http://localhost:8080/api');
    });
  });

  describe('#loginUser', () => {
    it('sets the bearer token', (done) => {
      const request = validLoginRequest();

      Kolide.loginUser({
        username: 'admin',
        password: 'secret',
      })
        .then(() => {
          expect(request.isDone()).toEqual(true);
          expect(Kolide.bearerToken).toEqual('auth_token');
          done();
        })
        .catch(done);
    });
  });
});
