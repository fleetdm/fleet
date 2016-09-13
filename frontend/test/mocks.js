import nock from 'nock';

export const validLoginRequest = () => {
  return nock('http://localhost:8080')
  .post('/api/v1/kolide/login')
  .reply(200, {
    token: 'auth_token',
    id: 1,
    username: 'admin',
    email: 'admin@kolide.co',
    name: '',
    admin: true,
    enabled: true,
    needs_password_reset: false,
  });
};

export default {
  validLoginRequest,
};
