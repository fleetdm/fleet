import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { noop } from 'lodash';
import { push } from 'react-router-redux';
import { resetPassword } from '../../redux/nodes/components/ResetPasswordPage/actions';
import ResetPasswordForm from '../../components/forms/ResetPasswordForm';
import StackedWhiteBoxes from '../../components/StackedWhiteBoxes';

export class ResetPasswordPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    token: PropTypes.string,
  };

  static defaultProps = {
    dispatch: noop,
  };

  componentWillMount () {
    const { dispatch, token } = this.props;

    if (!token) return dispatch(push('/login'));

    return false;
  }

  onSubmit = (formData) => {
    const { dispatch, token } = this.props;
    const resetPasswordData = {
      ...formData,
      password_reset_token: token,
    };

    return dispatch(resetPassword(resetPasswordData))
      .then(() => {
        return dispatch(push('/login'));
      });
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

  return {
    ...componentState,
    token,
  };
};

export default connect(mapStateToProps)(ResetPasswordPage);
