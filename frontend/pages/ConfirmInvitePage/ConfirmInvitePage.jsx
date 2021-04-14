import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { push } from "react-router-redux";

import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import ConfirmInviteForm from "components/forms/ConfirmInviteForm";
import EnsureUnauthenticated from "components/EnsureUnauthenticated";
import paths from "router/paths";
import { renderFlash } from "redux/nodes/notifications/actions";
import userActions from "redux/nodes/entities/users/actions";

const baseClass = "confirm-invite-page";

class ConfirmInvitePage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    inviteFormData: PropTypes.shape({
      email: PropTypes.string.isRequired,
      invite_token: PropTypes.string.isRequired,
      name: PropTypes.string.isRequired,
    }).isRequired,
    userErrors: PropTypes.shape({
      base: PropTypes.string,
    }),
  };

  componentWillUnmount() {
    const { dispatch } = this.props;
    const { clearErrors } = userActions;

    dispatch(clearErrors());

    return false;
  }

  onSubmit = (formData) => {
    const { create } = userActions;
    const { dispatch } = this.props;
    const { LOGIN } = paths;

    dispatch(create(formData))
      .then(() => {
        dispatch(push(LOGIN));
        dispatch(
          renderFlash(
            "success",
            "Registration successful! For security purposes, please log in."
          )
        );
      })
      .catch(() => false);

    return false;
  };

  render() {
    const { inviteFormData, userErrors } = this.props;
    const { onSubmit } = this;

    return (
      <AuthenticationFormWrapper>
        <div className={`${baseClass}`}>
          <div className={`${baseClass}__lead-wrapper`}>
            <p className={`${baseClass}__lead-text`}>Welcome to Fleet</p>
            <p className={`${baseClass}__sub-lead-text`}>
              Before you get started, please take a moment to complete the
              following information.
            </p>
          </div>
          <ConfirmInviteForm
            className={`${baseClass}__form`}
            formData={inviteFormData}
            handleSubmit={onSubmit}
            serverErrors={userErrors}
          />
        </div>
      </AuthenticationFormWrapper>
    );
  }
}

const mapStateToProps = (state, { location: urlLocation, params }) => {
  const { email, name } = urlLocation.query;
  const { invite_token: inviteToken } = params;
  const inviteFormData = { email, invite_token: inviteToken, name };
  const { errors: userErrors } = state.entities.users;

  return { inviteFormData, userErrors };
};

const ConnectedComponent = connect(mapStateToProps)(ConfirmInvitePage);
export default EnsureUnauthenticated(ConnectedComponent);
