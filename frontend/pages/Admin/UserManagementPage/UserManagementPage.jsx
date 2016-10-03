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

  onUserActionSelect = (user, formData) => {
    const { dispatch } = this.props;
    const { update } = userActions;

    if (formData.user_actions) {
      switch (formData.user_actions) {
        case 'demote_user':
          return dispatch(update(user, { admin: false }))
            .then(() => {
              return dispatch(renderFlash('success', 'User demoted', update(user, { admin: true })));
            });
        case 'disable_account':
          return dispatch(userActions.update(user, { enabled: false }))
            .then(() => {
              return dispatch(renderFlash('success', 'User account disabled', update(user, { enabled: true })));
            });
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
        case 'edit_user':
          return dispatch(update(user, formData.updated_user))
            .then(() => {
              return dispatch(renderFlash('success', 'User updated', update(user, user)));
            });
        default:
          return false;
      }
    }

    return false;
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
    const { onUserActionSelect } = this;

    return (
      <UserBlock
        key={user.email}
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
  const { entities: users } = entityGetter(state).get('users');

  return { users };
};

export default connect(mapStateToProps)(UserManagementPage);

