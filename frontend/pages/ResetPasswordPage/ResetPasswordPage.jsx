import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { noop } from 'lodash';
import { push } from 'react-router-redux';
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
    console.log('ResetPasswordForm data', formData);
    return false;
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

  return {
    token,
  };
};

export default connect(mapStateToProps)(ResetPasswordPage);
