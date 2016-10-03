import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { noop } from 'lodash';
import { push } from 'react-router-redux';
import debounce from '../../utilities/debounce';
import { resetPassword } from '../../redux/nodes/components/ResetPasswordPage/actions';
import ResetPasswordForm from '../../components/forms/ResetPasswordForm';
import StackedWhiteBoxes from '../../components/StackedWhiteBoxes';
import userActions from '../../redux/nodes/entities/users/actions';

export class ResetPasswordPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    token: PropTypes.string,
    user: PropTypes.object,
  };

  static defaultProps = {
    dispatch: noop,
  };

  componentWillMount () {
    const { dispatch, token, user } = this.props;

    if (!user && !token) return dispatch(push('/login'));

    return false;
  }

  onSubmit = debounce((formData) => {
    const { dispatch, token, user } = this.props;

    if (user) return this.updateUser(formData);

    const resetPasswordData = {
      ...formData,
      password_reset_token: token,
    };

    return dispatch(resetPassword(resetPasswordData))
      .then(() => {
        return dispatch(push('/login'));
      });
  })

  updateUser = (formData) => {
    const { dispatch, user } = this.props;
    const { new_password: password } = formData;
    const passwordUpdateParams = { password };

    return dispatch(userActions.update(user, passwordUpdateParams))
      .then(() => { return dispatch(push('/')); });
  }

  render () {
    const { onSubmit } = this;

    return (
      <StackedWhiteBoxes
        headerText="Reset Password"
        leadText="Create a new password using at least one letter, one numeral and seven characters."
      >
        <ResetPasswordForm onSubmit={onSubmit} />
      </StackedWhiteBoxes>
    );
  }
}

const mapStateToProps = (state, ownProps) => {
  const { query = {} } = ownProps.location || {};
  const { token } = query;
  const { ResetPasswordPage: componentState } = state.components;
  const { user } = state.auth;

  return {
    ...componentState,
    token,
    user,
  };
};

export default connect(mapStateToProps)(ResetPasswordPage);
