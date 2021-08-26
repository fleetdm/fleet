import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { size } from "lodash";
import { push } from "react-router-redux";

import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import {
  clearAuthErrors,
  loginUser,
  ssoRedirect,
} from "redux/nodes/auth/actions";
import { clearRedirectLocation } from "redux/nodes/redirectLocation/actions";
import debounce from "utilities/debounce";
import LoginForm from "components/forms/LoginForm";
import LoginSuccessfulPage from "pages/LoginSuccessfulPage";
import ForgotPasswordPage from "pages/ForgotPasswordPage";
import ResetPasswordPage from "pages/ResetPasswordPage";
import paths from "router/paths";
import redirectLocationInterface from "interfaces/redirect_location";
import userInterface from "interfaces/user";
import ssoSettingsInterface from "interfaces/ssoSettings";

export class LoginPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    errors: PropTypes.shape({
      base: PropTypes.string,
    }),
    pathname: PropTypes.string,
    isForgotPassPage: PropTypes.bool,
    isResetPassPage: PropTypes.bool,
    token: PropTypes.string,
    redirectLocation: redirectLocationInterface,
    user: userInterface,
    ssoSettings: ssoSettingsInterface,
  };

  constructor(props) {
    super(props);
    this.state = {
      loginVisible: true,
    };
  }

  componentWillMount() {
    const { dispatch, pathname, user } = this.props;
    const { HOME, LOGIN } = paths;

    if (user && pathname === LOGIN) {
      return dispatch(push(HOME));
    }

    return false;
  }

  onChange = () => {
    const { dispatch, errors } = this.props;

    if (size(errors)) {
      return dispatch(clearAuthErrors);
    }

    return false;
  };

  onSubmit = debounce((formData) => {
    const { dispatch, redirectLocation } = this.props;
    const { HOME } = paths;
    const redirectTime = 1500;
    return dispatch(loginUser(formData))
      .then((user) => {
        this.setState({ loginVisible: false });

        // Redirect to password reset page if user is forced to reset password.
        // Any other requests will fail.
        if (user.force_password_reset) {
          return dispatch(push(paths.RESET_PASSWORD));
        }

        setTimeout(() => {
          const nextLocation = redirectLocation || HOME;
          dispatch(clearRedirectLocation);
          return dispatch(push(nextLocation));
        }, redirectTime);
      })
      .catch(() => false);
  });

  ssoSignOn = () => {
    const { dispatch, redirectLocation } = this.props;
    const { HOME } = paths;
    let returnToAfterAuth = HOME;
    if (redirectLocation != null) {
      returnToAfterAuth = redirectLocation.pathname;
    }

    dispatch(ssoRedirect(returnToAfterAuth))
      .then((result) => {
        window.location.href = result.payload.ssoRedirectURL;
      })
      .catch(() => false);
  };

  showLoginForm = () => {
    const { errors, ssoSettings } = this.props;
    const { loginVisible } = this.state;
    const { onChange, onSubmit, ssoSignOn } = this;

    return (
      <LoginForm
        onChangeFunc={onChange}
        handleSubmit={onSubmit}
        isHidden={!loginVisible}
        serverErrors={errors}
        ssoSettings={ssoSettings}
        handleSSOSignOn={ssoSignOn}
      />
    );
  };

  render() {
    const { showLoginForm } = this;
    const { isForgotPassPage, isResetPassPage, token } = this.props;

    return (
      <AuthenticationFormWrapper>
        <LoginSuccessfulPage />
        {showLoginForm()}
        {isForgotPassPage && <ForgotPasswordPage />}
        {isResetPassPage && <ResetPasswordPage token={token} />}
      </AuthenticationFormWrapper>
    );
  }
}

const mapStateToProps = (state) => {
  const { errors, loading, user, ssoSettings } = state.auth;
  const { redirectLocation } = state;

  return {
    errors,
    loading,
    redirectLocation,
    user,
    ssoSettings,
  };
};

export default connect(mapStateToProps)(LoginPage);
