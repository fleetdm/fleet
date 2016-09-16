import expect from 'expect';
import Kolide from './index';
import mocks from '../test/mocks';

const {
  invalidPasswordResetRequest,
  validLoginRequest,
  validPasswordResetRequest,
  validUser,
} = mocks;

describe('Kolide - API client', () => {
  describe('defaults', () => {
    it('sets the base URL', () => {
      expect(Kolide.baseURL).toEqual('http://localhost:8080/api');
    });
  });

  describe('#loginUser', () => {
    it('calls the appropriate endpoint with the correct parameters', (done) => {
      const request = validLoginRequest();

      Kolide.loginUser({
        username: 'admin',
        password: 'secret',
      })
        .then((user) => {
          expect(user).toEqual(validUser);
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });
  });

  describe('#forgotPassword', () => {
    it('calls the appropriate endpoint with the correct parameters when successful', (done) => {
      const request = validPasswordResetRequest();
      const email = 'hi@thegnar.co';

      Kolide.forgotPassword({ email })
        .then(() => {
          expect(request.isDone()).toEqual(true);
          done();
        })
        .catch(done);
    });

    it('return errors correctly for unsuccessful requests', (done) => {
      const error = 'Something went wrong';
      const request = invalidPasswordResetRequest(error);
      const email = 'hi@thegnar.co';

      Kolide.forgotPassword({ email })
        .then(done)
        .catch(errorResponse => {
          const { response } = errorResponse;

          expect(response).toEqual({ error });
          expect(request.isDone()).toEqual(true);
          done();
        });
    });
  });
});
