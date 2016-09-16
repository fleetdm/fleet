import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';
import componentStyles from './styles';
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

  onSubmit = (formData) => {
    const { dispatch } = this.props;
    return dispatch(loginUser(formData))
      .then(() => {
        return dispatch(push('/login_successful'));
      });
  }

  render () {
    const { formWrapperStyles, whiteTabStyles } = componentStyles;
    const { onSubmit } = this;

    return (
      <div style={formWrapperStyles}>
        <div style={whiteTabStyles} />
        <LoginForm onSubmit={onSubmit} />
      </div>
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
