import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';

import componentStyles from './styles';
import entityGetter from '../../../redux/utilities/entityGetter';
import Button from '../../../components/buttons/Button';
import inviteActions from '../../../redux/nodes/entities/invites/actions';
import InviteUserForm from '../../../components/forms/InviteUserForm';
import Modal from '../../../components/Modal';
import userActions from '../../../redux/nodes/entities/users/actions';
import UserBlock from './UserBlock';
import { renderFlash } from '../../../redux/nodes/notifications/actions';

class UserManagementPage extends Component {
  static propTypes = {
    currentUser: PropTypes.object,
    dispatch: PropTypes.func,
    invites: PropTypes.arrayOf(PropTypes.object),
    users: PropTypes.arrayOf(PropTypes.object),
  };

  constructor (props) {
    super(props);

    this.state = {
      showInviteUserModal: false,
    };
  }

  componentWillMount () {
    const { dispatch, invites, users } = this.props;

    if (!users.length) {
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
          return dispatch(inviteActions.destroy({ entityID: user.id }))
            .then(() => {
              return dispatch(renderFlash('success', 'Invite revoked'));
            });
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
        return dispatch(
          renderFlash('success', 'User updated', update(user, user))
        );
      });
  }

  onInviteUserSubmit = (formData) => {
    const { dispatch } = this.props;

    dispatch(inviteActions.create(formData))
      .then(() => {
        dispatch(renderFlash('success', 'User invited'));
        return this.toggleInviteUserModal();
      })
      .catch(error => {
        const inviteError = error === 'resource already created'
          ? 'User has already been invited'
          : error;
        this.setState({ inviteError });
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

  renderUserBlock = (user, options = { invite: false }) => {
    const { currentUser } = this.props;
    const { invite } = options;
    const { onEditUser, onUserActionSelect } = this;

    return (
      <UserBlock
        currentUser={currentUser}
        invite={invite}
        key={user.email}
        onEditUser={onEditUser}
        onSelect={onUserActionSelect}
        user={user}
      />
    );
  }

  renderModal = () => {
    const { currentUser } = this.props;
    const { inviteError, showInviteUserModal } = this.state;
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
          error={inviteError}
          invitedBy={currentUser}
          onCancel={onInviteCancel}
          onSubmit={onInviteUserSubmit}
        />
      </Modal>
    );
  };

  render () {
    const {
      addUserButtonStyles,
      addUserWrapperStyles,
      containerStyles,
      numUsersStyles,
      usersWrapperStyles,
    } = componentStyles;
    const { toggleInviteUserModal } = this;
    const { invites, users } = this.props;
    const resourcesCount = users.length + invites.length;

    return (
      <div style={containerStyles}>
        <span style={numUsersStyles}>Listing {resourcesCount} users</span>
        <div style={addUserWrapperStyles}>
          <Button
            onClick={toggleInviteUserModal}
            style={addUserButtonStyles}
            text="Add User"
          />
        </div>
        <div style={usersWrapperStyles}>
          {users.map(user => {
            return this.renderUserBlock(user);
          })}
          {invites.map(user => {
            return this.renderUserBlock(user, { invite: true });
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

  return { currentUser, invites, users };
};

export default connect(mapStateToProps)(UserManagementPage);

