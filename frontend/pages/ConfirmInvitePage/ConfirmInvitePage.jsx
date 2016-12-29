import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { join, map } from 'lodash';
import { push } from 'react-router-redux';

import AuthenticationFormWrapper from 'components/AuthenticationFormWrapper';
import ConfirmInviteForm from 'components/forms/ConfirmInviteForm';
import EnsureUnauthenticated from 'components/EnsureUnauthenticated';
import paths from 'router/paths';
import { renderFlash } from 'redux/nodes/notifications/actions';
import userActions from 'redux/nodes/entities/users/actions';

const baseClass = 'confirm-invite-page';

class ConfirmInvitePage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    inviteFormData: PropTypes.shape({
      email: PropTypes.string.isRequired,
      invite_token: PropTypes.string.isRequired,
      name: PropTypes.string.isRequired,
    }).isRequired,
  };

  onSubmit = (formData) => {
    const { create } = userActions;
    const { dispatch } = this.props;
    const { LOGIN } = paths;

    dispatch(create(formData))
      .then(() => {
        dispatch(push(LOGIN));
        dispatch(renderFlash('success', 'Registration successful! For security purposes, please log in.'));
      })
      .catch((errors) => {
        const errorMessages = map(errors, (error) => {
          return `${error.name}: ${error.reason}`;
        });
        const formattedErrorMessage = join(errorMessages, ',');

        dispatch(renderFlash('error', formattedErrorMessage));

        return false;
      });

    return false;
  }

  render () {
    const { inviteFormData } = this.props;
    const { onSubmit } = this;

    return (
      <AuthenticationFormWrapper>
        <div className={`${baseClass}__lead-wrapper`}>
          <p className={`${baseClass}__lead-text`}>
            Welcome to the party, {inviteFormData.email}!
          </p>
          <p className={`${baseClass}__sub-lead-text`}>
            Please take a moment to fill out the following information before we take you into <b>Kolide</b>
          </p>
        </div>
        <div className={`${baseClass}__form-section-wrapper`}>
          <div className={`${baseClass}__form-section-description`}>
            <h2>SET USERNAME & PASSWORD</h2>
            <p>Password must include 7 characters, at least 1 number (eg. 0-9), and at least 1 symbol (eg. ^&*#)</p>
          </div>
          <ConfirmInviteForm
            className={`${baseClass}__form`}
            formData={inviteFormData}
            handleSubmit={onSubmit}
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

  return { inviteFormData };
};

const ConnectedComponent = connect(mapStateToProps)(ConfirmInvitePage);
export default EnsureUnauthenticated(ConnectedComponent);
