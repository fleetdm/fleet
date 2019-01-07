import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Avatar from 'components/Avatar';
import Dropdown from 'components/forms/fields/Dropdown';
import EditUserForm from 'components/forms/admin/EditUserForm';
import helpers from 'components/UserBlock/helpers';
import userInterface from 'interfaces/user';

class UserBlock extends Component {
  static propTypes = {
    isCurrentUser: PropTypes.bool.isRequired,
    isEditing: PropTypes.bool.isRequired,
    isInvite: PropTypes.bool.isRequired,
    onEditUser: PropTypes.func.isRequired,
    onToggleEditUser: PropTypes.func.isRequired,
    onSelect: PropTypes.func,
    user: userInterface.isRequired,
    userErrors: PropTypes.shape({
      base: PropTypes.string,
      name: PropTypes.string,
      username: PropTypes.string,
    }),
  };

  onToggleEditing = (evt) => {
    evt.preventDefault();

    const { onToggleEditUser, user } = this.props;

    return onToggleEditUser(user);
  }

  onEditUser = (updatedUser) => {
    const { onEditUser, user } = this.props;

    return onEditUser(user, updatedUser);
  }

  onUserActionSelect = (action) => {
    const { onSelect, onToggleEditUser, user } = this.props;

    if (action === 'modify_details') {
      return onToggleEditUser(user);
    }

    return onSelect(user, action);
  }

  renderCTAs = () => {
    const { isCurrentUser, isInvite, user } = this.props;
    const { onUserActionSelect } = this;
    const userActionOptions = helpers.userActionOptions(isCurrentUser, user, isInvite);

    return (
      <Dropdown
        name="user-action-dropdown"
        options={userActionOptions}
        placeholder="Actions..."
        onChange={onUserActionSelect}
        className={isInvite ? 'revoke-invite' : ''}
      />
    );
  }

  render () {
    const { isEditing, isInvite, user, userErrors } = this.props;
    const {
      admin,
      email,
      name,
      position,
      username,
      enabled,
    } = user;
    const { onEditUser, onToggleEditing, renderCTAs } = this;
    const statusLabel = helpers.userStatusLabel(user, isInvite);
    const userLabel = admin ? 'Admin' : 'User';

    const baseClass = 'user-block';

    const userWrapperClass = classnames(
      baseClass,
      { [`${baseClass}--invited`]: isInvite },
      { [`${baseClass}--disabled`]: !enabled && !isInvite }
    );

    const userHeaderClass = classnames(
      `${baseClass}__header`,
      { [`${baseClass}__header--admin`]: admin },
      { [`${baseClass}__header--user`]: !admin },
      { [`${baseClass}__header--invited`]: isInvite },
      { [`${baseClass}__header--disabled`]: !enabled && !isInvite }
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
      { [`${baseClass}__status-text--invited`]: isInvite },
      { [`${baseClass}__status-text--enabled`]: enabled },
      { [`${baseClass}__status-text--disabled`]: !enabled && !isInvite }
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

    if (isEditing) {
      return (
        <div className={userWrapperClass}>
          <EditUserForm
            onCancel={onToggleEditing}
            handleSubmit={onEditUser}
            formData={user}
            serverErrors={userErrors}
          />
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
