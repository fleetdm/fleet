import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';

import Button from 'components/buttons/Button';
import configInterface from 'interfaces/config';
import deepDifference from 'utilities/deep_difference';
import entityGetter from 'redux/utilities/entityGetter';
import inviteActions from 'redux/nodes/entities/invites/actions';
import inviteInterface from 'interfaces/invite';
import InviteUserForm from 'components/forms/InviteUserForm';
import Modal from 'components/modals/Modal';
import paths from 'router/paths';
import SmtpWarning from 'components/SmtpWarning';
import Spinner from 'components/loaders/Spinner';
import userActions from 'redux/nodes/entities/users/actions';
import userInterface from 'interfaces/user';
import { renderFlash } from 'redux/nodes/notifications/actions';
import UserBlock from './UserBlock';

const baseClass = 'user-management';

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

  constructor (props) {
    super(props);

    this.state = {
      showInviteUserModal: false,
    };
  }

  componentWillMount () {
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
        case 'demote_user': {
          if (currentUser.id === user.id) {
            return dispatch(renderFlash('error', 'You cannot demote yourself'));
          }
          return dispatch(updateAdmin(user, { admin: false }))
            .then(() => {
              return dispatch(renderFlash('success', 'User demoted', updateAdmin(user, { admin: true })));
            });
        }
        case 'disable_account': {
          if (currentUser.id === user.id) {
            return dispatch(renderFlash('error', 'You cannot disable your own account'));
          }
          return dispatch(userActions.enableUser(user, { enabled: false }))
            .then(() => {
              return dispatch(renderFlash('success', 'User account disabled', enableUser(user, { enabled: true })));
            });
        }
        case 'enable_account':
          return dispatch(enableUser(user, { enabled: true }))
            .then(() => {
              return dispatch(renderFlash('success', 'User account enabled', enableUser(user, { enabled: false })));
            });
        case 'promote_user':
          return dispatch(updateAdmin(user, { admin: true }))
            .then(() => {
              return dispatch(renderFlash('success', 'User promoted to admin', updateAdmin(user, { admin: false })));
            });
        case 'reset_password':
          return dispatch(requirePasswordReset(user, { require: true }))
            .then(() => {
              return dispatch(renderFlash('success', 'User required to reset password', requirePasswordReset(user, { require: false })));
            });
        case 'revert_invitation':
          return dispatch(inviteActions.destroy(user))
            .then(() => dispatch(renderFlash('success', 'Invite revoked')))
            .catch(() => dispatch(renderFlash('error', 'Invite could not be revoked')));
        default:
          return false;
      }
    }

    return false;
  }

  onEditUser = (user, updatedUser) => {
    const { dispatch } = this.props;
    const { update } = userActions;
    const updatedAttrs = deepDifference(updatedUser, user);

    return dispatch(update(user, updatedAttrs))
      .then(() => {
        dispatch(renderFlash('success', 'User updated', update(user, user)));

        return Promise.resolve();
      });
  }

  onInviteUserSubmit = (formData) => {
    const { dispatch } = this.props;

    dispatch(inviteActions.create(formData))
      .then(() => {
        dispatch(renderFlash('success', 'User invited'));

        return this.toggleInviteUserModal();
      })
      .catch(() => false);
  }

  onInviteCancel = (evt) => {
    evt.preventDefault();

    return this.toggleInviteUserModal();
  }

  goToAppConfigPage = (evt) => {
    evt.preventDefault();

    const { ADMIN_SETTINGS } = paths;
    const { dispatch } = this.props;

    dispatch(push(ADMIN_SETTINGS));
  }

  toggleInviteUserModal = () => {
    const { showInviteUserModal } = this.state;

    this.setState({
      showInviteUserModal: !showInviteUserModal,
    });

    return false;
  }

  renderUserBlock = (user, idx, options = { invite: false }) => {
    const { currentUser, userErrors } = this.props;
    const { invite } = options;
    const { onEditUser, onUserActionSelect } = this;

    return (
      <UserBlock
        currentUser={currentUser}
        invite={invite}
        key={`${user.email}-${idx}-${invite ? 'invite' : 'user'}`}
        onEditUser={onEditUser}
        onSelect={onUserActionSelect}
        user={user}
        userErrors={userErrors}
      />
    );
  }

  renderModal = () => {
    const { currentUser, inviteErrors } = this.props;
    const { showInviteUserModal } = this.state;
    const { onInviteCancel, onInviteUserSubmit, toggleInviteUserModal } = this;

    if (!showInviteUserModal) {
      return false;
    }

    return (
      <Modal
        title="Invite New User"
        onExit={toggleInviteUserModal}
        className={`${baseClass}__invite-modal`}
      >
        <InviteUserForm
          serverErrors={inviteErrors}
          invitedBy={currentUser}
          onCancel={onInviteCancel}
          onSubmit={onInviteUserSubmit}
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
        <SmtpWarning
          onResolve={goToAppConfigPage}
          shouldShowWarning={!config.configured}
        />
      </div>
    );
  }

  renderUsersAndInvites = () => {
    const { invites, loadingInvites, loadingUsers, users } = this.props;
    const { renderUserBlock } = this;

    if (loadingInvites || loadingUsers) {
      return <div className={`${baseClass}__users`}><Spinner /></div>;
    }

    return (
      <div className={`${baseClass}__users`}>
        {users.map((user, idx) => {
          return renderUserBlock(user, idx);
        })}
        {invites.map((user, idx) => {
          return renderUserBlock(user, idx, { invite: true });
        })}
      </div>
    );
  }

  render () {
    const { renderModal, renderSmtpWarning, renderUsersAndInvites, toggleInviteUserModal } = this;
    const { config, invites, users } = this.props;
    const resourcesCount = users.length + invites.length;

    return (
      <div className={`${baseClass} body-wrap`}>
        <div className={`${baseClass}__heading-wrapper`}>
          <h1 className={`${baseClass}__user-count`}>Listing {resourcesCount} users</h1>
          <div className={`${baseClass}__add-user-wrap`}>
            <Button
              className={`${baseClass}__add-user-btn`}
              disabled={!config.configured}
              onClick={toggleInviteUserModal}
            >
              Add User
            </Button>
          </div>
        </div>
        {renderSmtpWarning()}
        {renderUsersAndInvites()}
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
  const { entities: users } = stateEntityGetter.get('users');
  const { entities: invites } = stateEntityGetter.get('invites');
  const { errors: inviteErrors, loading: loadingInvites } = state.entities.invites;
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
