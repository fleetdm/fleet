import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { noop, size } from "lodash";
import { push } from "react-router-redux";

import debounce from "utilities/debounce";
import {
  clearResetPasswordErrors,
  resetPassword,
} from "redux/nodes/components/ResetPasswordPage/actions";
import ResetPasswordForm from "components/forms/ResetPasswordForm";
import StackedWhiteBoxes from "components/StackedWhiteBoxes";
import { performRequiredPasswordReset } from "redux/nodes/auth/actions";
import userInterface from "interfaces/user";
import PATHS from "router/paths";

export class ResetPasswordPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    errors: PropTypes.shape({
      base: PropTypes.string,
      new_password: PropTypes.string,
    }),
    token: PropTypes.string,
    user: userInterface,
  };

  static defaultProps = {
    dispatch: noop,
  };

  componentWillMount() {
    const { dispatch, token, user } = this.props;

    if (!user && !token) {
      return dispatch(push(PATHS.LOGIN));
    }

    return false;
  }

  onResetErrors = () => {
    const { dispatch, errors } = this.props;

    if (size(errors)) {
      dispatch(clearResetPasswordErrors);
    }

    return false;
  };

  onSubmit = debounce((formData) => {
    const { dispatch, token, user } = this.props;

    if (user) {
      return this.loggedInUser(formData);
    }

    const resetPasswordData = {
      ...formData,
      password_reset_token: token,
    };

    return dispatch(resetPassword(resetPasswordData))
      .then(() => {
        return dispatch(push(PATHS.LOGIN));
      })
      .catch(() => false);
  });

  handleLeave = (location) => {
    const { dispatch } = this.props;

    return dispatch(push(location));
  };

  loggedInUser = (formData) => {
    const { dispatch } = this.props;
    const { new_password: password } = formData;
    const passwordUpdateParams = { password };

    return dispatch(performRequiredPasswordReset(passwordUpdateParams))
      .then(() => {
        return dispatch(push(PATHS.HOME));
      })
      .catch(() => false);
  };

  render() {
    const { handleLeave, onResetErrors, onSubmit } = this;
    const { errors } = this.props;

    return (
      <StackedWhiteBoxes
        leadText="Create a new password. Your new password must include 7 characters, at least 1 number (e.g. 0 - 9), and at least 1 symbol (e.g. &*#)"
        onLeave={handleLeave}
      >
        <ResetPasswordForm
          handleSubmit={onSubmit}
          onChangeFunc={onResetErrors}
          serverErrors={errors}
        />
      </StackedWhiteBoxes>
    );
  }
}

const mapStateToProps = (state) => {
  const { ResetPasswordPage: componentState } = state.components;
  const { user, errors } = state.auth;

  return {
    ...componentState,
    user,
    errors,
  };
};

export default connect(mapStateToProps)(ResetPasswordPage);
