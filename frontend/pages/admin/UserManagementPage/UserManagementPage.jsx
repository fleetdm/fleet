import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { concat, includes, difference } from "lodash";
import { push } from "react-router-redux";

import Button from "components/buttons/Button";
import configInterface from "interfaces/config";
import deepDifference from "utilities/deep_difference";
import entityGetter from "redux/utilities/entityGetter";
import inviteActions from "redux/nodes/entities/invites/actions";
import inviteInterface from "interfaces/invite";
import InviteUserForm from "components/forms/InviteUserForm";
import Modal from "components/modals/Modal";
import paths from "router/paths";
import { renderFlash } from "redux/nodes/notifications/actions";
import WarningBanner from "components/WarningBanner";
import { updateUser } from "redux/nodes/auth/actions";
import userActions from "redux/nodes/entities/users/actions";
import UserRow from "components/UserRow";
import userInterface from "interfaces/user";

const baseClass = "user-management";

export class UserManagementPage extends Component {
  static propTypes = {
    appConfigLoading: PropTypes.bool,
    config: configInterface,
    currentUser: userInterface,
    dispatch: PropTypes.func,
    inviteErrors: PropTypes.shape({
      base: PropTypes.string,
      email: PropTypes.string,
    }),
    invites: PropTypes.arrayOf(inviteInterface),
    loadingInvites: PropTypes.bool,
    loadingUsers: PropTypes.bool,
    userErrors: PropTypes.shape({
      base: PropTypes.string,
      name: PropTypes.string,
      username: PropTypes.string,
    }),
    users: PropTypes.arrayOf(userInterface),
  };

  constructor(props) {
    super(props);

    this.state = {
      showInviteUserModal: false,
      usersEditing: [],
    };
  }

  componentWillMount() {
    const { dispatch } = this.props;

    dispatch(userActions.loadAll());
    dispatch(inviteActions.loadAll());

    return false;
  }

  onUserActionSelect = (user, action) => {
    const { currentUser, dispatch } = this.props;
    const { enableUser, updateAdmin, requirePasswordReset } = userActions;

    if (action) {
      switch (action) {
        case "demote_user": {
          if (currentUser.id === user.id) {
            return dispatch(renderFlash("error", "You cannot demote yourself"));
          }
          return dispatch(updateAdmin(user, { admin: false })).then(() => {
            return dispatch(
              renderFlash(
                "success",
                "User demoted",
                updateAdmin(user, { admin: true })
              )
            );
          });
        }
        case "disable_account": {
          if (currentUser.id === user.id) {
            return dispatch(
              renderFlash("error", "You cannot disable your own account")
            );
          }
          return dispatch(
            userActions.enableUser(user, { enabled: false })
          ).then(() => {
            return dispatch(
              renderFlash(
                "success",
                "User account disabled",
                enableUser(user, { enabled: true })
              )
            );
          });
        }
        case "enable_account":
          return dispatch(enableUser(user, { enabled: true })).then(() => {
            return dispatch(
              renderFlash(
                "success",
                "User account enabled",
                enableUser(user, { enabled: false })
              )
            );
          });
        case "promote_user":
          return dispatch(updateAdmin(user, { admin: true })).then(() => {
            return dispatch(
              renderFlash(
                "success",
                "User promoted to admin",
                updateAdmin(user, { admin: false })
              )
            );
          });
        case "reset_password":
          return dispatch(requirePasswordReset(user, { require: true })).then(
            () => {
              return dispatch(
                renderFlash(
                  "success",
                  "User required to reset password",
                  requirePasswordReset(user, { require: false })
                )
              );
            }
          );
        case "revert_invitation":
          return dispatch(inviteActions.silentDestroy(user))
            .then(() => dispatch(renderFlash("success", "Invite revoked")))
            .catch(() =>
              dispatch(renderFlash("error", "Invite could not be revoked"))
            );
        default:
          return false;
      }
    }

    return false;
  };

  onEditUser = (user, updatedUser) => {
    const { currentUser, dispatch } = this.props;
    const { onToggleEditUser } = this;
    const { silentUpdate } = userActions;
    const updatedAttrs = deepDifference(updatedUser, user);

    if (currentUser.id === user.id) {
      return dispatch(updateUser(user, updatedAttrs))
        .then(() => {
          dispatch(
            renderFlash("success", "User updated", updateUser(user, user))
          );
          onToggleEditUser(user);

          return false;
        })
        .catch(() => false);
    }

    return dispatch(silentUpdate(user, updatedAttrs))
      .then(() => {
        dispatch(
          renderFlash("success", "User updated", silentUpdate(user, user))
        );
        onToggleEditUser(user);

        return false;
      })
      .catch(() => false);
  };

  onInviteUserSubmit = (formData) => {
    const { dispatch } = this.props;

    dispatch(inviteActions.silentCreate(formData))
      .then(() => {
        return this.toggleInviteUserModal();
      })
      .catch(() => false);
  };

  onInviteCancel = (evt) => {
    evt.preventDefault();

    return this.toggleInviteUserModal();
  };

  onToggleEditUser = (user) => {
    const { dispatch } = this.props;
    const { usersEditing } = this.state;
    let updatedUsersEditing = [];

    dispatch(userActions.clearErrors());

    if (includes(usersEditing, user.id)) {
      updatedUsersEditing = difference(usersEditing, [user.id]);
    } else {
      updatedUsersEditing = concat(usersEditing, [user.id]);
    }

    this.setState({ usersEditing: updatedUsersEditing });
  };

  goToAppConfigPage = (evt) => {
    evt.preventDefault();

    const { ADMIN_SETTINGS } = paths;
    const { dispatch } = this.props;

    dispatch(push(ADMIN_SETTINGS));
  };

  toggleInviteUserModal = () => {
    const { showInviteUserModal } = this.state;

    this.setState({
      showInviteUserModal: !showInviteUserModal,
    });

    return false;
  };

  renderUserRow = (user, idx, options = { invite: false }) => {
    const { currentUser, userErrors } = this.props;
    const { invite } = options;
    const { onEditUser, onToggleEditUser, onUserActionSelect } = this;
    const { usersEditing } = this.state;
    const isEditing = includes(usersEditing, user.id);

    return (
      <UserRow
        isEditing={isEditing}
        isInvite={invite}
        isCurrentUser={currentUser.id === user.id}
        key={`${user.email}-${idx}-${invite ? "invite" : "user"}`}
        onEditUser={onEditUser}
        onSelect={onUserActionSelect}
        onToggleEditUser={onToggleEditUser}
        user={user}
        userErrors={userErrors}
      />
    );
  };

  renderModal = () => {
    const { currentUser, inviteErrors } = this.props;
    const { showInviteUserModal } = this.state;
    const { onInviteCancel, onInviteUserSubmit, toggleInviteUserModal } = this;
    const ssoEnabledForApp = this.props.config.enable_sso;

    if (!showInviteUserModal) {
      return false;
    }

    return (
      <Modal
        title="Invite new user"
        onExit={toggleInviteUserModal}
        className={`${baseClass}__invite-modal`}
      >
        <InviteUserForm
          serverErrors={inviteErrors}
          invitedBy={currentUser}
          onCancel={onInviteCancel}
          onSubmit={onInviteUserSubmit}
          canUseSSO={ssoEnabledForApp}
        />
      </Modal>
    );
  };

  renderSmtpWarning = () => {
    const { appConfigLoading, config } = this.props;
    const { goToAppConfigPage } = this;

    if (appConfigLoading) {
      return false;
    }

    return (
      <div className={`${baseClass}__smtp-warning-wrapper`}>
        <WarningBanner shouldShowWarning={!config.configured}>
          <span>
            SMTP is not currently configured in Fleet. The &quot;Invite
            user&quot; feature requires that SMTP is configured in order to send
            invitation emails.
          </span>
          <Button
            className={`${baseClass}__config-button`}
            onClick={goToAppConfigPage}
            variant={"unstyled"}
          >
            Configure SMTP
          </Button>
        </WarningBanner>
      </div>
    );
  };

  renderUserTable = () => {
    const { invites, users } = this.props;
    const { renderUserRow } = this;

    return (
      <div className={`${baseClass}__wrapper`}>
        <table className={`${baseClass}__table`}>
          <thead>
            <tr>
              <th>Username</th>
              <th>Status</th>
              <th>Full Name</th>
              <th>Email</th>
              <th>Role</th>
              <th className={`${baseClass}__position`}>Position</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {users.map((user, idx) => {
              return renderUserRow(user, idx);
            })}
            {invites.map((user, idx) => {
              return renderUserRow(user, idx, { invite: true });
            })}
          </tbody>
        </table>
      </div>
    );
  };

  render() {
    const {
      renderModal,
      renderSmtpWarning,
      renderUserTable,
      toggleInviteUserModal,
    } = this;
    const { config, loadingInvites, loadingUsers, users, invites } = this.props;
    const resourcesCount = users.length + invites.length;
    if (loadingInvites || loadingUsers) {
      return false;
    }

    return (
      <div className={`${baseClass} body-wrap`}>
        <p className={`${baseClass}__page-description`}>
          Invite new users, customize user permissions, and disable users in
          Fleet.
        </p>
        {renderSmtpWarning()}
        <div className={`${baseClass}__add-user-wrap`}>
          <p className={`${baseClass}__user-count`}>{resourcesCount} users</p>
          <Button
            className={"button button--brand"}
            disabled={!config.configured}
            onClick={toggleInviteUserModal}
            title={
              config.configured
                ? "Add User"
                : "Email must be configured to add users"
            }
          >
            Invite user
          </Button>
        </div>
        {renderUserTable()}
        {renderModal()}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const stateEntityGetter = entityGetter(state);
  const { config } = state.app;
  const { loading: appConfigLoading } = state.app;
  const { user: currentUser } = state.auth;
  const { entities: users } = stateEntityGetter.get("users");
  const { entities: invites } = stateEntityGetter.get("invites");
  const {
    errors: inviteErrors,
    loading: loadingInvites,
  } = state.entities.invites;
  const { errors: userErrors, loading: loadingUsers } = state.entities.users;

  return {
    appConfigLoading,
    config,
    currentUser,
    inviteErrors,
    invites,
    loadingInvites,
    loadingUsers,
    userErrors,
    users,
  };
};

export default connect(mapStateToProps)(UserManagementPage);
