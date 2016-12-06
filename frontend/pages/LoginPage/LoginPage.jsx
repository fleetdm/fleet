import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { includes } from 'lodash';
import { push } from 'react-router-redux';

import AuthenticationFormWrapper from 'components/AuthenticationFormWrapper';
import { clearAuthErrors, loginUser } from 'redux/nodes/auth/actions';
import { clearRedirectLocation } from 'redux/nodes/redirectLocation/actions';
import debounce from 'utilities/debounce';
import LoginForm from 'components/forms/LoginForm';
import LoginSuccessfulPage from 'pages/LoginSuccessfulPage';
import ForgotPasswordPage from 'pages/ForgotPasswordPage';
import ResetPasswordPage from 'pages/ResetPasswordPage';
import paths from 'router/paths';
import redirectLocationInterface from 'interfaces/redirect_location';
import userInterface from 'interfaces/user';

const WHITELIST_ERRORS = ['Unable to authenticate the current user'];

export class LoginPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    error: PropTypes.string,
    pathname: PropTypes.string,
    isForgotPassPage: PropTypes.bool,
    isResetPassPage: PropTypes.bool,
    token: PropTypes.string,
    redirectLocation: redirectLocationInterface,
    user: userInterface,
  };

  constructor (props) {
    super(props);
    this.state = {
      loginVisible: true,
    };
  }

  componentWillMount () {
    const { dispatch, pathname, user } = this.props;
    const { HOME, LOGIN } = paths;

    if (user && pathname === LOGIN) {
      return dispatch(push(HOME));
    }

    return false;
  }

  onChange = () => {
    const { dispatch, error } = this.props;

    if (error) {
      return dispatch(clearAuthErrors);
    }

    return false;
  };

  onSubmit = debounce((formData) => {
    const { dispatch, redirectLocation } = this.props;
    const { HOME } = paths;
    const redirectTime = 1500;
    return dispatch(loginUser(formData))
      .then(() => {
        this.setState({ loginVisible: false });
        setTimeout(() => {
          const nextLocation = redirectLocation || HOME;
          dispatch(clearRedirectLocation);
          return dispatch(push(nextLocation));
        }, redirectTime);
      });
  })

  serverErrors = () => {
    const { error } = this.props;

    if (!error || includes(WHITELIST_ERRORS, error)) {
      return {};
    }

    return {
      username: error,
      password: 'password',
    };
  }

  showLoginForm = () => {
    const { loginVisible } = this.state;
    const { onChange, onSubmit, serverErrors } = this;

    return (
      <LoginForm
        onChangeFunc={onChange}
        handleSubmit={onSubmit}
        isHidden={!loginVisible}
        errors={serverErrors()}
      />
    );
  }

  render () {
    const { showLoginForm } = this;
    const { isForgotPassPage, isResetPassPage, token } = this.props;

    return (
      <AuthenticationFormWrapper>
        <LoginSuccessfulPage />
        {showLoginForm()}
        { isForgotPassPage && <ForgotPasswordPage /> }
        { isResetPassPage && <ResetPasswordPage token={token} /> }
      </AuthenticationFormWrapper>
    );
  }
}

const mapStateToProps = (state) => {
  const { error, loading, user } = state.auth;
  const { redirectLocation } = state;

  return {
    error,
    loading,
    redirectLocation,
    user,
  };
};

export default connect(mapStateToProps)(LoginPage);
