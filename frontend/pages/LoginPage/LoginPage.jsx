import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';
import AuthenticationFormWrapper from '../../components/AuthenticationFormWrapper';
import debounce from '../../utilities/debounce';
import local from '../../utilities/local';
import LoginForm from '../../components/forms/LoginForm';
import { loginUser } from '../../redux/nodes/auth/actions';

export class LoginPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    error: PropTypes.string,
    loading: PropTypes.bool,
    user: PropTypes.object,
  };

  componentWillMount () {
    const { dispatch } = this.props;

    if (local.getItem('auth_token')) {
      return dispatch(push('/'));
    }

    return false;
  }

  onSubmit = debounce((formData) => {
    const { dispatch } = this.props;
    return dispatch(loginUser(formData))
      .then(() => {
        return dispatch(push('/login_successful'));
      });
  })

  render () {
    const { onSubmit } = this;

    return (
      <AuthenticationFormWrapper>
        <LoginForm onSubmit={onSubmit} />
      </AuthenticationFormWrapper>
    );
  }
}

const mapStateToProps = (state) => {
  const { error, loading, user } = state.auth;

  return {
    error,
    loading,
    user,
  };
};

export default connect(mapStateToProps)(LoginPage);
