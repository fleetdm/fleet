import nock from 'nock';

export const validUser = {
  token: 'auth_token',
  id: 1,
  username: 'admin',
  email: 'admin@kolide.co',
  name: '',
  admin: true,
  enabled: true,
  needs_password_reset: false,
};

export const validLoginRequest = () => {
  return nock('http://localhost:8080')
  .post('/api/v1/kolide/login')
  .reply(200, validUser);
};

export const validPasswordResetRequest = () => {
  return nock('http://localhost:8080')
  .post('/api/v1/kolide/forgot_password')
  .reply(200, validUser);
};

export const invalidPasswordResetRequest = (error) => {
  return nock('http://localhost:8080')
  .post('/api/v1/kolide/forgot_password')
  .reply(422, { error });
};

export default {
  invalidPasswordResetRequest,
  validLoginRequest,
  validPasswordResetRequest,
  validUser,
};
