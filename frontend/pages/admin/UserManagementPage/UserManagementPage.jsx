import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { isEqual } from "lodash";
import { push } from "react-router-redux";
import memoize from "memoize-one";

import TableContainer from "components/TableContainer";
import Modal from "components/modals/Modal";
import inviteInterface from "interfaces/invite";
import configInterface from "interfaces/config";
import userInterface from "interfaces/user";
import teamInterface from "interfaces/team";
import permissionUtils from "utilities/permissions";
import paths from "router/paths";
import entityGetter from "redux/utilities/entityGetter";
import inviteActions from "redux/nodes/entities/invites/actions";
import { renderFlash } from "redux/nodes/notifications/actions";
import { updateUser } from "redux/nodes/auth/actions";
import userActions from "redux/nodes/entities/users/actions";
import teamActions from "redux/nodes/entities/teams/actions";

import UserForm from "./components/UserForm";
import EmptyUsers from "./components/EmptyUsers";
import { generateTableHeaders, combineDataSets } from "./UsersTableConfig";
import DeleteUserForm from "./components/DeleteUserForm";
import ResetPasswordModal from "./components/ResetPasswordModal";
import ResetSessionsModal from "./components/ResetSessionsModal";
import { NewUserType } from "./components/UserForm/UserForm";

const baseClass = "user-management";

const generateUpdateData = (currentUserData, formData) => {
  const updatableFields = [
    "global_role",
    "teams",
    "name",
    "email",
    "sso_enabled",
  ];
  return Object.keys(formData).reduce((updatedAttributes, attr) => {
    // attribute can be updated and is different from the current value.
    if (
      updatableFields.includes(attr) &&
      !isEqual(formData[attr], currentUserData[attr])
    ) {
      updatedAttributes[attr] = formData[attr];
    }
    return updatedAttributes;
  }, {});
};

export class UserManagementPage extends Component {
  static propTypes = {
    appConfigLoading: PropTypes.bool,
    config: configInterface,
    currentUser: userInterface,
    dispatch: PropTypes.func,
    loadingTableData: PropTypes.bool,
    invites: PropTypes.arrayOf(inviteInterface),
    inviteErrors: PropTypes.shape({
      base: PropTypes.string,
      email: PropTypes.string,
    }),
    isBasicTier: PropTypes.bool,
    users: PropTypes.arrayOf(userInterface),
    userErrors: PropTypes.shape({
      base: PropTypes.string,
      name: PropTypes.string,
    }),
    teams: PropTypes.arrayOf(teamInterface),
  };

  constructor(props) {
    super(props);

    this.state = {
      showCreateUserModal: false,
      showEditUserModal: false,
      showDeleteUserModal: false,
      showResetPasswordModal: false,
      showResetSessionsModal: false,
      userEditing: null,
      usersEditing: [],
    };
  }

  componentDidMount() {
    const { dispatch, isBasicTier } = this.props;
    if (isBasicTier) {
      dispatch(teamActions.loadAll({}));
    }
  }

  onEditUser = (formData) => {
    const { currentUser, config, dispatch } = this.props;
    const { userEditing } = this.state;
    const { toggleEditUserModal, getUser } = this;

    const userData = getUser(userEditing.type, userEditing.id);

    const updatedAttrs = generateUpdateData(userData, formData);
    if (currentUser.id === userEditing.id) {
      return dispatch(updateUser(userData, updatedAttrs))
        .then(() => {
          dispatch(
            renderFlash("success", `Successfully edited ${userEditing?.name}`)
          );
          toggleEditUserModal();
        })
        .catch(() => {
          dispatch(
            renderFlash(
              "error",
              `Could not edit ${userEditing?.name}. Please try again.`
            )
          );
          toggleEditUserModal();
        });
    }

    let userUpdatedFlashMessage = `Successfully edited ${formData.name}`;

    if (userData.email !== formData.email) {
      userUpdatedFlashMessage += `: A confirmation email was sent from ${config.sender_address} to ${formData.email}`;
    }

    return dispatch(userActions.silentUpdate(userData, formData))
      .then(() => {
        dispatch(renderFlash("success", userUpdatedFlashMessage));
        toggleEditUserModal();
      })
      .catch(() => {
        dispatch(
          renderFlash(
            "error",
            `Couldn not edit ${userEditing?.name}. Please try again.`
          )
        );
        toggleEditUserModal();
      });
  };

  onCreateUserSubmit = (formData) => {
    const { dispatch, config } = this.props;

    if (formData.newUserType === NewUserType.AdminInvited) {
      // Do some data formatting adding `invited_by` for the request to be correct and deleteing uncessary fields
      const requestData = {
        ...formData,
        invited_by: formData.currentUserId,
      };
      delete requestData.currentUserId; // this field is not needed for the request
      delete requestData.newUserType; // this field is not needed for the request
      delete requestData.password; // this field is not needed for the request
      dispatch(inviteActions.create(requestData))
        .then(() => {
          dispatch(
            renderFlash(
              "success",
              `An invitation email was sent from ${config.sender_address} to ${formData.email}.`
            )
          );
          this.toggleCreateUserModal();
        })
        .catch(() => {
          dispatch(
            renderFlash("error", "Could not create user. Please try again.")
          );
          this.toggleCreateUserModal();
        });
    } else {
      // Do some data formatting deleteing uncessary fields
      const requestData = {
        ...formData,
      };
      delete requestData.currentUserId; // this field is not needed for the request
      delete requestData.newUserType; // this field is not needed for the request
      dispatch(userActions.createUserWithoutInvitation(requestData))
        .then(() => {
          dispatch(
            renderFlash("success", `Successfully created ${requestData.name}.`)
          );
          this.toggleCreateUserModal();
        })
        .catch(() => {
          dispatch(
            renderFlash("error", "Could not create user. Please try again.")
          );
          this.toggleCreateUserModal();
        });
    }
  };

  onCreateCancel = (evt) => {
    evt.preventDefault();
    this.toggleCreateUserModal();
  };

  onDeleteUser = () => {
    const { dispatch } = this.props;
    const { userEditing } = this.state;
    const { toggleDeleteUserModal } = this;

    if (userEditing.type === "invite") {
      dispatch(inviteActions.destroy(userEditing))
        .then(() => {
          dispatch(
            renderFlash("success", `Successfully deleted ${userEditing?.name}.`)
          );
        })
        .catch(() => {
          dispatch(
            renderFlash(
              "error",
              `Could not delete ${userEditing?.name}. Please try again.`
            )
          );
        });
      toggleDeleteUserModal();
    } else {
      dispatch(userActions.destroy(userEditing))
        .then(() => {
          dispatch(
            renderFlash("success", `Successfully deleted ${userEditing?.name}.`)
          );
        })
        .catch(() => {
          dispatch(
            renderFlash(
              "error",
              `Could not delete ${userEditing?.name}. Please try again.`
            )
          );
        });
      toggleDeleteUserModal();
    }
  };

  onResetSessions = () => {
    const { LOGIN } = paths;
    const { currentUser, dispatch } = this.props;
    const { userEditing } = this.state;
    const { toggleResetSessionsUserModal } = this;
    dispatch(userActions.deleteSessions(userEditing))
      .then(() => {
        if (currentUser.id === userEditing.id) {
          dispatch(push(LOGIN));
        } else {
          dispatch(renderFlash("success", "Sessions reset"));
        }
      })
      .catch(() => {
        dispatch(
          renderFlash(
            "error",
            "Could not reset sessions for the selected user. Please try again."
          )
        );
      });
    toggleResetSessionsUserModal();
  };

  // NOTE: this is called once on the initial rendering. The initial render of
  // the TableContainer child component calls this handler.
  onTableQueryChange = (queryData) => {
    const { dispatch } = this.props;
    const {
      pageIndex,
      pageSize,
      searchQuery,
      sortHeader,
      sortDirection,
    } = queryData;
    let sortBy = [];
    if (sortHeader !== "") {
      sortBy = [{ id: sortHeader, direction: sortDirection }];
    }
    dispatch(
      userActions.loadAll({
        page: pageIndex,
        perPage: pageSize,
        globalFilter: searchQuery,
        sortBy,
      })
    );
    dispatch(inviteActions.loadAll(pageIndex, pageSize, searchQuery, sortBy));
  };

  onActionSelect = (action, user) => {
    const {
      toggleEditUserModal,
      toggleDeleteUserModal,
      goToUserSettingsPage,
      toggleResetPasswordUserModal,
      toggleResetSessionsUserModal,
    } = this;
    switch (action) {
      case "edit":
        toggleEditUserModal(user);
        break;
      case "delete":
        toggleDeleteUserModal(user);
        break;
      case "passwordReset":
        toggleResetPasswordUserModal(user);
        break;
      case "resetSessions":
        toggleResetSessionsUserModal(user);
        break;
      case "editMyAccount":
        goToUserSettingsPage();
        break;
      default:
        return null;
    }
    return null;
  };

  getUser = (type, id) => {
    const { users, invites } = this.props;
    let userData;
    if (type === "user") {
      userData = users.find((user) => user.id === id);
    } else {
      userData = invites.find((invite) => invite.id === id);
    }
    return userData;
  };

  toggleCreateUserModal = () => {
    const { showCreateUserModal } = this.state;
    this.setState({
      showCreateUserModal: !showCreateUserModal,
    });
  };

  toggleEditUserModal = (user) => {
    const { showEditUserModal } = this.state;
    this.setState({
      showEditUserModal: !showEditUserModal,
      userEditing: !showEditUserModal ? user : null,
    });
  };

  toggleDeleteUserModal = (user) => {
    const { showDeleteUserModal } = this.state;
    this.setState({
      showDeleteUserModal: !showDeleteUserModal,
      userEditing: !showDeleteUserModal ? user : null,
    });
  };

  toggleResetPasswordUserModal = (user) => {
    const { showResetPasswordModal } = this.state;
    this.setState({
      showResetPasswordModal: !showResetPasswordModal,
      userEditing: !showResetPasswordModal ? user : null,
    });
  };

  toggleResetSessionsUserModal = (user) => {
    const { showResetSessionsModal } = this.state;
    this.setState({
      showResetSessionsModal: !showResetSessionsModal,
      userEditing: !showResetSessionsModal ? user : null,
    });
  };

  combineUsersAndInvites = memoize((users, invites, currentUserId) => {
    return combineDataSets(users, invites, currentUserId);
  });

  resetPassword = (user) => {
    const { dispatch } = this.props;
    const { toggleResetPasswordUserModal } = this;
    const { requirePasswordReset } = userActions;

    return dispatch(requirePasswordReset(user.id, { require: true })).then(
      () => {
        dispatch(
          renderFlash(
            "success",
            "User required to reset password",
            requirePasswordReset(user.id, { require: false }) // this is an undo action.
          )
        );
        toggleResetPasswordUserModal();
      }
    );
  };

  goToUserSettingsPage = () => {
    const { USER_SETTINGS } = paths;
    const { dispatch } = this.props;

    dispatch(push(USER_SETTINGS));
  };

  goToAppConfigPage = (evt) => {
    evt.preventDefault();

    const { ADMIN_SETTINGS } = paths;
    const { dispatch } = this.props;

    dispatch(push(ADMIN_SETTINGS));
  };

  renderEditUserModal = () => {
    const {
      currentUser,
      inviteErrors,
      config,
      teams,
      isBasicTier,
    } = this.props;
    const { showEditUserModal, userEditing } = this.state;
    const { onEditUser, toggleEditUserModal, getUser } = this;

    if (!showEditUserModal) return null;

    const userData = getUser(userEditing.type, userEditing.id);

    return (
      <Modal
        title="Edit user"
        onExit={toggleEditUserModal}
        className={`${baseClass}__edit-user-modal`}
      >
        <UserForm
          serverErrors={inviteErrors}
          defaultEmail={userData.email}
          defaultName={userData.name}
          defaultGlobalRole={userData.global_role}
          defaultTeams={userData.teams}
          currentUserId={currentUser.id}
          onCancel={toggleEditUserModal}
          onSubmit={onEditUser}
          availableTeams={teams}
          submitText={"Save"}
          isBasicTier={isBasicTier}
          smtpConfigured={config.configured}
          canUseSso={config.enable_sso}
          isSsoEnabled={userData.sso_enabled}
        />
      </Modal>
    );
  };

  renderCreateUserModal = () => {
    const {
      currentUser,
      inviteErrors,
      config,
      teams,
      isBasicTier,
    } = this.props;
    const { showCreateUserModal } = this.state;
    const { onCreateUserSubmit, toggleCreateUserModal } = this;

    if (!showCreateUserModal) return null;

    return (
      <Modal
        title="Create user"
        onExit={toggleCreateUserModal}
        className={`${baseClass}__create-user-modal`}
      >
        <UserForm
          serverErrors={inviteErrors}
          currentUserId={currentUser.id}
          onCancel={toggleCreateUserModal}
          onSubmit={onCreateUserSubmit}
          availableTeams={teams}
          defaultGlobalRole={"observer"}
          defaultTeams={[]}
          defaultNewUserType={false}
          submitText={"Create"}
          isBasicTier={isBasicTier}
          smtpConfigured={config.configured}
          canUseSso={config.enable_sso}
          isNewUser
        />
      </Modal>
    );
  };

  renderDeleteUserModal = () => {
    const { showDeleteUserModal, userEditing } = this.state;
    const { toggleDeleteUserModal, onDeleteUser } = this;

    if (!showDeleteUserModal) return null;

    return (
      <Modal
        title={"Delete user"}
        onExit={toggleDeleteUserModal}
        className={`${baseClass}__delete-user-modal`}
      >
        <DeleteUserForm
          name={userEditing.name}
          onDelete={onDeleteUser}
          onCancel={toggleDeleteUserModal}
        />
      </Modal>
    );
  };

  renderResetPasswordModal = () => {
    const { showResetPasswordModal, userEditing } = this.state;
    const { toggleResetPasswordUserModal, resetPassword } = this;

    if (!showResetPasswordModal) return null;

    return (
      <ResetPasswordModal
        user={userEditing}
        modalBaseClass={baseClass}
        onResetConfirm={resetPassword}
        onResetCancel={toggleResetPasswordUserModal}
      />
    );
  };

  renderResetSessionsModal = () => {
    const { showResetSessionsModal, userEditing } = this.state;
    const { toggleResetSessionsUserModal, onResetSessions } = this;

    if (!showResetSessionsModal) return null;

    return (
      <ResetSessionsModal
        user={userEditing}
        modalBaseClass={baseClass}
        onResetConfirm={onResetSessions}
        onResetCancel={toggleResetSessionsUserModal}
      />
    );
  };

  render() {
    const {
      renderCreateUserModal,
      renderEditUserModal,
      renderDeleteUserModal,
      renderResetPasswordModal,
      renderResetSessionsModal,
      toggleCreateUserModal,
      onTableQueryChange,
      onActionSelect,
    } = this;

    const {
      loadingTableData,
      users,
      invites,
      currentUser,
      isBasicTier,
    } = this.props;

    const tableHeaders = generateTableHeaders(onActionSelect, isBasicTier);

    let tableData = [];
    if (!loadingTableData) {
      tableData = this.combineUsersAndInvites(
        users,
        invites,
        currentUser.id,
        onActionSelect
      );
    }

    return (
      <div className={`${baseClass} body-wrap`}>
        <p className={`${baseClass}__page-description`}>
          Create new users, customize user permissions, and remove users from
          Fleet.
        </p>
        {/* TODO: find a way to move these controls into the table component */}
        <TableContainer
          columns={tableHeaders}
          data={tableData}
          isLoading={loadingTableData}
          defaultSortHeader={"name"}
          defaultSortDirection={"asc"}
          inputPlaceHolder={"Search"}
          actionButtonText={"Create user"}
          onActionButtonClick={toggleCreateUserModal}
          onQueryChange={onTableQueryChange}
          resultsTitle={"users"}
          emptyComponent={EmptyUsers}
          searchable
        />
        {renderCreateUserModal()}
        {renderEditUserModal()}
        {renderDeleteUserModal()}
        {renderResetSessionsModal()}
        {renderResetPasswordModal()}
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const stateEntityGetter = entityGetter(state);
  const { config } = state.app;
  const { loading: appConfigLoading } = state.app;
  const { user: currentUser } = state.auth;
  const { entities: users } = stateEntityGetter.get("users");
  const { entities: invites } = stateEntityGetter.get("invites");
  const { entities: teams } = stateEntityGetter.get("teams");
  const {
    errors: inviteErrors,
    loading: loadingInvites,
  } = state.entities.invites;
  const { errors: userErrors, loading: loadingUsers } = state.entities.users;
  const loadingTableData = loadingUsers || loadingInvites;
  const isBasicTier = permissionUtils.isBasicTier(config);

  return {
    appConfigLoading,
    config,
    currentUser,
    users,
    userErrors,
    invites,
    inviteErrors,
    isBasicTier,
    loadingTableData,
    teams,
  };
};

export default connect(mapStateToProps)(UserManagementPage);
