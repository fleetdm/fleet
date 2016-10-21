import React, { Component, PropTypes } from 'react';
import radium from 'radium';

import Avatar from '../../../../components/Avatar';
import componentStyles from './styles';
import Dropdown from '../../../../components/forms/fields/Dropdown';
import EditUserForm from '../../../../components/forms/Admin/EditUserForm';
import userInterface from '../../../../interfaces/user';
import { userStatusLabel } from './helpers';

class UserBlock extends Component {
  static propTypes = {
    currentUser: userInterface,
    invite: PropTypes.bool,
    onEditUser: PropTypes.func,
    onSelect: PropTypes.func,
    user: userInterface,
  };

  static userActionOptions = (currentUser, user, invite) => {
    const disableActions = currentUser.id === user.id;
    const inviteActions = [
      { text: 'Actions...', value: '' },
      { text: 'Revoke Invitation', value: 'revert_invitation' },
    ];
    const userEnableAction = user.enabled
      ? { disabled: disableActions, text: 'Disable Account', value: 'disable_account' }
      : { text: 'Enable Account', value: 'enable_account' };
    const userPromotionAction = user.admin
      ? { disabled: disableActions, text: 'Demote User', value: 'demote_user' }
      : { text: 'Promote User', value: 'promote_user' };

    if (invite) return inviteActions;

    return [
      { text: 'Actions...', value: '' },
      userEnableAction,
      userPromotionAction,
      { text: 'Require Password Reset', value: 'reset_password' },
      { text: 'Modify Details', value: 'modify_details' },
    ];
  };

  constructor (props) {
    super(props);

    this.state = {
      isEdit: false,
    };
  }

  onToggleEditing = (evt) => {
    evt.preventDefault();

    const { isEdit } = this.state;

    this.setState({
      isEdit: !isEdit,
    });

    return false;
  }

  onEditUserFormSubmit = (updatedUser) => {
    const { user, onEditUser } = this.props;

    this.setState({
      isEdit: false,
    });

    return onEditUser(user, updatedUser);
  }

  onUserActionSelect = ({ target }) => {
    const { onSelect, user } = this.props;
    const { value: action } = target;

    if (action === 'modify_details') {
      this.setState({
        isEdit: true,
      });

      return false;
    }

    return onSelect(user, action);
  }

  renderCTAs = () => {
    const { currentUser, invite, user } = this.props;
    const { onUserActionSelect } = this;
    const userActionOptions = UserBlock.userActionOptions(currentUser, user, invite);
    const { revokeInviteStyles } = componentStyles(user, invite);

    return (
      <Dropdown
        containerStyles={invite ? revokeInviteStyles : {}}
        options={userActionOptions}
        initialOption={{ text: 'Actions...' }}
        onSelect={onUserActionSelect}
      />
    );
  }

  render () {
    const { invite, user } = this.props;
    const {
      avatarStyles,
      nameStyles,
      userDetailsStyles,
      userEmailStyles,
      userHeaderStyles,
      userLabelStyles,
      usernameStyles,
      userPositionStyles,
      userStatusStyles,
      userStatusWrapperStyles,
      userWrapperStyles,
    } = componentStyles(user, invite);
    const {
      admin,
      email,
      name,
      position,
      username,
    } = user;
    const { isEdit } = this.state;
    const { onEditUserFormSubmit, onToggleEditing, renderCTAs } = this;
    const statusLabel = userStatusLabel(user, invite);
    const userLabel = admin ? 'Admin' : 'User';

    if (isEdit) {
      return (
        <div style={userWrapperStyles}>
          <EditUserForm onCancel={onToggleEditing} onSubmit={onEditUserFormSubmit} user={user} />
        </div>
      );
    }

    return (
      <div style={userWrapperStyles}>
        <div style={userHeaderStyles}>
          <span style={nameStyles}>{name}</span>
        </div>
        <div style={userDetailsStyles}>
          <Avatar user={user} style={avatarStyles} />
          <div style={userStatusWrapperStyles}>
            <span style={userLabelStyles}>{userLabel}</span>
            <span style={userStatusStyles}>{statusLabel}</span>
            <div style={{ clear: 'both' }} />
          </div>
          <p style={usernameStyles}>{username}</p>
          <p style={userPositionStyles}>{position}</p>
          <p style={userEmailStyles}>{email}</p>
          {renderCTAs()}
        </div>
      </div>
    );
  }
}

export default radium(UserBlock);
