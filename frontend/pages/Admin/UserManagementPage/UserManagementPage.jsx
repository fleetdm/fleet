import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import componentStyles from './styles';
import entityGetter from '../../../redux/entityGetter';
import Button from '../../../components/buttons/Button';
import InviteUserForm from '../../../components/forms/InviteUserForm';
import Modal from '../../../components/Modal';
import userActions from '../../../redux/nodes/entities/users/actions';
import UserBlock from './UserBlock';
import { renderFlash } from '../../../redux/nodes/notifications/actions';

class UserManagementPage extends Component {
  static propTypes = {
    currentUser: PropTypes.object,
    dispatch: PropTypes.func,
    users: PropTypes.arrayOf(PropTypes.object),
  };

  constructor (props) {
    super(props);

    this.state = {
      showInviteUserModal: false,
    };
  }

  componentWillMount () {
    const { dispatch, users } = this.props;
    const { load } = userActions;

    if (!users.length) dispatch(load());

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
    console.log('user invited', formData);
    return this.toggleInviteUserModal();
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

  renderUserBlock = (user) => {
    const { currentUser } = this.props;
    const { onEditUser, onUserActionSelect } = this;

    return (
      <UserBlock
        currentUser={currentUser}
        key={user.email}
        onEditUser={onEditUser}
        onSelect={onUserActionSelect}
        user={user}
      />
    );
  }

  renderModal = () => {
    const { showInviteUserModal } = this.state;
    const { onInviteCancel, onInviteUserSubmit, toggleInviteUserModal } = this;

    if (!showInviteUserModal) return false;

    return (
      <Modal
        title="Invite new user"
        onExit={toggleInviteUserModal}
      >
        <InviteUserForm
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
    const { users } = this.props;

    return (
      <div style={containerStyles}>
        <span style={numUsersStyles}>Listing {users.length} users</span>
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
        </div>
        {this.renderModal()}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const { user: currentUser } = state.auth;
  const { entities: users } = entityGetter(state).get('users');

  return { currentUser, users };
};

export default connect(mapStateToProps)(UserManagementPage);

