import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import ConfirmSSOInviteForm from "components/forms/ConfirmSSOInviteForm";
import EnsureUnauthenticated from "components/EnsureUnauthenticated";
import userActions from "redux/nodes/entities/users/actions";
import authActions from "redux/nodes/auth/actions";
import paths from "router/paths";

const baseClass = "confirm-invite-page";

class ConfirmSSOInvitePage extends Component {
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
    const { ssoRedirect } = authActions;
    const { dispatch } = this.props;
    const { HOME } = paths;

    formData.sso_invite = true;
    dispatch(create(formData))
      .then(() => {
        // set redirect so that we will get redirected to home page after
        // the user authenticates with the idp
        dispatch(ssoRedirect(HOME))
          .then((result) => {
            window.location.href = result.payload.ssoRedirectURL;
          })
          .catch(() => false);
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
          <ConfirmSSOInviteForm
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

const ConnectedComponent = connect(mapStateToProps)(ConfirmSSOInvitePage);
export default EnsureUnauthenticated(ConnectedComponent);
