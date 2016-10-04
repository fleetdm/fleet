import React, { Component, PropTypes } from 'react';
import radium from 'radium';
import Avatar from '../../../../components/Avatar';
import componentStyles from './styles';
import Dropdown from '../../../../components/forms/fields/Dropdown';
import EditUserForm from '../../../../components/forms/Admin/EditUserForm';

class UserBlock extends Component {
  static propTypes = {
    onEditUser: PropTypes.func,
    onSelect: PropTypes.func,
    user: PropTypes.object,
  };

  static userActionOptions = (user) => {
    const userEnableAction = user.enabled
      ? { text: 'Disable Account', value: 'disable_account' }
      : { text: 'Enable Account', value: 'enable_account' };
    const userPromotionAction = user.admin
      ? { text: 'Demote User', value: 'demote_user' }
      : { text: 'Promote User', value: 'promote_user' };

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

  render () {
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
    } = componentStyles;
    const { user } = this.props;
    const {
      admin,
      email,
      enabled,
      name,
      position,
      username,
    } = user;
    const userLabel = admin ? 'Admin' : 'User';
    const activeLabel = enabled ? 'Active' : 'Disabled';
    const userActionOptions = UserBlock.userActionOptions(user);
    const { isEdit } = this.state;
    const { onEditUserFormSubmit, onToggleEditing } = this;

    if (isEdit) {
      return <EditUserForm onCancel={onToggleEditing} onSubmit={onEditUserFormSubmit} user={user} />;
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
            <span style={userStatusStyles(enabled)}>{activeLabel}</span>
            <div style={{ clear: 'both' }} />
          </div>
          <p style={usernameStyles}>{username}</p>
          <p style={userPositionStyles}>{position}</p>
          <p style={userEmailStyles}>{email}</p>
          <Dropdown
            options={userActionOptions}
            initialOption={{ text: 'Actions...' }}
            onSelect={this.onUserActionSelect}
          />
        </div>
      </div>
    );
  }
}

export default radium(UserBlock);
