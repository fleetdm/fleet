import Base from './base';
import endpoints from './endpoints';

class Kolide extends Base {
  forgotPassword ({ email }) {
    const { FORGOT_PASSWORD } = endpoints;
    const forgotPasswordEndpoint = this.baseURL + FORGOT_PASSWORD;

    return this.post(forgotPasswordEndpoint, JSON.stringify({ email }));
  }

  loginUser ({ username, password }) {
    const { LOGIN } = endpoints;
    const loginEndpoint = this.baseURL + LOGIN;

    return this.post(loginEndpoint, JSON.stringify({ username, password }));
  }

  logout () {
    const { LOGOUT } = endpoints;
    const logoutEndpoint = this.baseURL + LOGOUT;

    return this.authenticatedPost(logoutEndpoint);
  }

  me () {
    const { ME } = endpoints;
    const meEndpoint = this.baseURL + ME;

    return this.authenticatedGet(meEndpoint);
  }

  resetPassword (formData) {
    const { RESET_PASSWORD } = endpoints;
    const resetPasswordEndpoint = this.baseURL + RESET_PASSWORD;

    return this.post(resetPasswordEndpoint, JSON.stringify(formData));
  }
}

export default new Kolide();
