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

export const validForgotPasswordRequest = () => {
  return nock('http://localhost:8080')
  .post('/api/v1/kolide/forgot_password')
  .reply(200, validUser);
};

export const invalidForgotPasswordRequest = (error) => {
  return nock('http://localhost:8080')
  .post('/api/v1/kolide/forgot_password')
  .reply(422, { error });
};

export const validResetPasswordRequest = (password, token) => {
  return nock('http://localhost:8080')
  .post('/api/v1/kolide/reset_password', JSON.stringify({
    new_password: password,
    password_reset_token: token,
  }))
  .reply(200, validUser);
};

export const invalidResetPasswordRequest = (password, token, error) => {
  return nock('http://localhost:8080')
  .post('/api/v1/kolide/reset_password', JSON.stringify({
    new_password: password,
    password_reset_token: token,
  }))
  .reply(422, { error });
};


export default {
  invalidForgotPasswordRequest,
  invalidResetPasswordRequest,
  validForgotPasswordRequest,
  validLoginRequest,
  validResetPasswordRequest,
  validUser,
};
