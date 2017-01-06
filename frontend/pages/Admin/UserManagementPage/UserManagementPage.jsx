import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { first, isEqual, size } from 'lodash';

import entityGetter from 'redux/utilities/entityGetter';
import Button from 'components/buttons/Button';
import inviteActions from 'redux/nodes/entities/invites/actions';
import inviteInterface from 'interfaces/invite';
import InviteUserForm from 'components/forms/InviteUserForm';
import Modal from 'components/modals/Modal';
import userActions from 'redux/nodes/entities/users/actions';
import userInterface from 'interfaces/user';
import { renderFlash } from 'redux/nodes/notifications/actions';
import UserBlock from './UserBlock';

class UserManagementPage extends Component {
  static propTypes = {
    currentUser: userInterface,
    dispatch: PropTypes.func,
    inviteErrors: PropTypes.shape({
      base: PropTypes.string,
      email: PropTypes.string,
    }),
    invites: PropTypes.arrayOf(inviteInterface),
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
    const { currentUser, dispatch, invites, users } = this.props;

    if (!size(users) ||
      (size(users) === 1 && isEqual(first(users), currentUser))) {
      dispatch(userActions.loadAll());
    }

    if (!invites.length) {
      dispatch(inviteActions.loadAll());
    }

    return false;
  }

  onUserActionSelect = (user, action) => {
    const { currentUser, dispatch } = this.props;
    const { update } = userActions;

    if (action) {
      switch (action) {
        case 'demote_user': {
          if (currentUser.id === user.id) {
            return dispatch(renderFlash('error', 'You cannot demote yourself'));
          }
          return dispatch(update(user, { admin: false }))
            .then(() => {
              return dispatch(renderFlash('success', 'User demoted', update(user, { admin: true })));
            });
        }
        case 'disable_account': {
          if (currentUser.id === user.id) {
            return dispatch(renderFlash('error', 'You cannot disable your own account'));
          }
          return dispatch(userActions.update(user, { enabled: false }))
            .then(() => {
              return dispatch(renderFlash('success', 'User account disabled', update(user, { enabled: true })));
            });
        }
        case 'enable_account':
          return dispatch(update(user, { enabled: true }))
            .then(() => {
              return dispatch(renderFlash('success', 'User account enabled', update(user, { enabled: false })));
            });
        case 'promote_user':
          return dispatch(update(user, { admin: true }))
            .then(() => {
              return dispatch(renderFlash('success', 'User promoted to admin', update(user, { admin: false })));
            });
        case 'reset_password':
          return dispatch(update(user, { force_password_reset: true }))
            .then(() => {
              return dispatch(renderFlash('success', 'User forced to reset password', update(user, { force_password_reset: false })));
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

    return dispatch(update(user, updatedUser))
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
      });
  }

  onInviteCancel = (evt) => {
    evt.preventDefault();

    return this.toggleInviteUserModal();
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
        title="Invite new user"
        onExit={toggleInviteUserModal}
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

  render () {
    const { toggleInviteUserModal } = this;
    const { invites, users } = this.props;
    const resourcesCount = users.length + invites.length;

    const baseClass = 'user-management';

    return (
      <div className={`${baseClass} body-wrap`}>
        <h1 className={`${baseClass}__user-count`}>Listing {resourcesCount} users</h1>
        <div className={`${baseClass}__add-user-wrap`}>
          <Button onClick={toggleInviteUserModal} className={`${baseClass}__add-user-btn`}>
            Add User
          </Button>
        </div>
        <div className={`${baseClass}__users`}>
          {users.map((user, idx) => {
            return this.renderUserBlock(user, idx);
          })}
          {invites.map((user, idx) => {
            return this.renderUserBlock(user, idx, { invite: true });
          })}
        </div>
        {this.renderModal()}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const stateEntityGetter = entityGetter(state);
  const { user: currentUser } = state.auth;
  const { entities: users } = stateEntityGetter.get('users');
  const { entities: invites } = stateEntityGetter.get('invites');
  const { errors: inviteErrors } = state.entities.invites;
  const { errors: userErrors } = state.entities.users;

  return { currentUser, inviteErrors, invites, userErrors, users };
};

export default connect(mapStateToProps)(UserManagementPage);

