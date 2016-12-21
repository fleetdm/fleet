import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';

import Avatar from '../../../../components/Avatar';
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
      { label: 'Revoke Invitation', value: 'revert_invitation' },
    ];
    const userEnableAction = user.enabled
      ? { disabled: disableActions, label: 'Disable Account', value: 'disable_account' }
      : { label: 'Enable Account', value: 'enable_account' };
    const userPromotionAction = user.admin
      ? { disabled: disableActions, label: 'Demote User', value: 'demote_user' }
      : { label: 'Promote User', value: 'promote_user' };

    if (invite) return inviteActions;

    return [
      userEnableAction,
      userPromotionAction,
      { label: 'Require Password Reset', value: 'reset_password' },
      { label: 'Modify Details', value: 'modify_details' },
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

  onUserActionSelect = (action) => {
    const { onSelect, user } = this.props;

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

    return (
      <Dropdown
        options={userActionOptions}
        placeholder="Actions..."
        onChange={onUserActionSelect}
        className={invite ? 'revoke-invite' : ''}
      />
    );
  }

  render () {
    const { invite, user } = this.props;
    const {
      admin,
      email,
      name,
      position,
      username,
      enabled,
    } = user;
    const { isEdit } = this.state;
    const { onEditUserFormSubmit, onToggleEditing, renderCTAs } = this;
    const statusLabel = userStatusLabel(user, invite);
    const userLabel = admin ? 'Admin' : 'User';

    const baseClass = 'user-block';

    const userWrapperClass = classnames(
      baseClass,
      { [`${baseClass}--invited`]: invite },
      { [`${baseClass}--disabled`]: !enabled && !invite }
    );

    const userHeaderClass = classnames(
      `${baseClass}__header`,
      { [`${baseClass}__header--admin`]: admin },
      { [`${baseClass}__header--user`]: !admin },
      { [`${baseClass}__header--invited`]: invite },
      { [`${baseClass}__header--disabled`]: !enabled && !invite }
    );

    const userAvatarClass = classnames(
      `${baseClass}__avatar`,
      { [`${baseClass}__avatar--enabled`]: enabled }
    );

    const userStatusLabelClass = classnames(
      `${baseClass}__status-label`,
      { [`${baseClass}__status-label--admin`]: admin }
    );

    const userStatusTextClass = classnames(
      `${baseClass}__status-text`,
      { [`${baseClass}__status-text--invited`]: invite },
      { [`${baseClass}__status-text--enabled`]: enabled },
      { [`${baseClass}__status-text--disabled`]: !enabled && !invite }
    );

    const userUsernameClass = classnames(
      `${baseClass}__username`,
      { [`${baseClass}__username--enabled`]: enabled },
      { [`${baseClass}__username--hidden`]: !username }
    );

    const userPositionClass = classnames(
      `${baseClass}__position`,
      { [`${baseClass}__position--hidden`]: !position }
    );

    const userEmailClass = classnames(
      `${baseClass}__email`,
      { [`${baseClass}__email--disabled`]: !enabled }
    );

    if (isEdit) {
      return (
        <div className={userWrapperClass}>
          <EditUserForm onCancel={onToggleEditing} handleSubmit={onEditUserFormSubmit} formData={user} />
        </div>
      );
    }

    return (
      <div className={userWrapperClass}>
        <div className={userHeaderClass}>
          <span className={`${baseClass}__header-name`}>{name}</span>
        </div>
        <div className={`${baseClass}__details`}>
          <Avatar user={user} className={userAvatarClass} />
          <div className={`${baseClass}__status-wrapper`}>
            <span className={userStatusLabelClass}>{userLabel}</span>
            <span className={userStatusTextClass}>{statusLabel}</span>
            <div className="cf" />
          </div>
          <p className={userUsernameClass}>{username}</p>
          <p className={userPositionClass} title={position}>{position}</p>
          <p className={userEmailClass}>{email}</p>
          {renderCTAs()}
        </div>
      </div>
    );
  }
}

export default UserBlock;
