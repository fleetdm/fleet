import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import Dropdown from "components/forms/fields/Dropdown";
import EditUserForm from "components/forms/admin/EditUserForm";
import Modal from "components/modals/Modal";
import helpers from "components/UserRow/helpers";
import userInterface from "interfaces/user";

class UserRow extends Component {
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
  };

  onEditUser = (updatedUser) => {
    const { onEditUser, user } = this.props;

    return onEditUser(user, updatedUser);
  };

  onUserActionSelect = (action) => {
    const { onSelect, onToggleEditUser, user } = this.props;

    if (action === "modify_details") {
      return onToggleEditUser(user);
    }

    return onSelect(user, action);
  };

  renderCTAs = () => {
    const { isCurrentUser, isInvite, user } = this.props;
    const { onUserActionSelect } = this;
    const userActionOptions = helpers.userActionOptions(
      isCurrentUser,
      user,
      isInvite
    );

    return (
      <Dropdown
        name="user-action-dropdown"
        options={userActionOptions}
        placeholder="Actions..."
        onChange={onUserActionSelect}
        className={isInvite ? "revoke-invite" : ""}
      />
    );
  };

  renderEditUserModal = (isEditing) => {
    const { userErrors, isCurrentUser, user } = this.props;
    const { onEditUser, onToggleEditing } = this;

    if (isEditing) {
      return (
        <Modal title="Edit user" onExit={onToggleEditing}>
          <EditUserForm
            isCurrentUser={isCurrentUser}
            onCancel={onToggleEditing}
            handleSubmit={onEditUser}
            formData={user}
            serverErrors={userErrors}
          />
        </Modal>
      );
    }
    return false;
  };

  render() {
    const { isInvite, user, isEditing } = this.props;
    const { admin, email, name, position, username } = user;
    const { renderCTAs, renderEditUserModal } = this;
    const statusLabel = helpers.userStatusLabel(user, isInvite);
    const userLabel = admin ? "Admin" : "User";

    const baseClass = "user-row";
    const statusClassName = classnames(
      `${baseClass}__status`,
      `${baseClass}__status--${statusLabel.toLowerCase()}`
    );

    return (
      <tr key={`user-${user.id}-table`}>
        <td className={`${baseClass}__username`}>{username}</td>
        <td className={statusClassName}>{statusLabel}</td>
        <td>{name}</td>
        <td>{email}</td>
        <td>{userLabel}</td>
        <td className={`${baseClass}__position`}>{position}</td>
        <td className={`${baseClass}__actions`}>
          {renderCTAs()}
          {renderEditUserModal(isEditing)}
        </td>
      </tr>
    );
  }
}

export default UserRow;
