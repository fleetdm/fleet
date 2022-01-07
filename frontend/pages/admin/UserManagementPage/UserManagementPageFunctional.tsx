import React, { useState, useCallback, Component } from "react";
import { connect, useDispatch } from "react-redux";
import { isEqual } from "lodash";
import { push } from "react-router-redux";
import memoize from "memoize-one";

// @ts-ignore
import Fleet from "fleet";
import { IInvite } from "interfaces/invite";
import { IConfig } from "interfaces/config";
import { IUser } from "interfaces/user";
import { ITeam } from "interfaces/team";
import permissionUtils from "utilities/permissions";
import paths from "router/paths";
import entityGetter from "redux/utilities/entityGetter";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
// @ts-ignore
import { updateUser } from "redux/nodes/auth/actions";
// @ts-ignore
import inviteActions from "redux/nodes/entities/invites/actions";
// @ts-ignore
import userActions from "redux/nodes/entities/users/actions";
import teamActions from "redux/nodes/entities/teams/actions";

import TableContainer from "components/TableContainer";
import TableDataError from "components/TableDataError";
import Modal from "components/Modal";
import { DEFAULT_CREATE_USER_ERRORS } from "utilities/constants";
import EmptyUsers from "./components/EmptyUsers";
import { generateTableHeaders, combineDataSets } from "./UsersTableConfig";
import DeleteUserForm from "./components/DeleteUserForm";
import ResetPasswordModal from "./components/ResetPasswordModal";
import ResetSessionsModal from "./components/ResetSessionsModal";
import { NewUserType } from "./components/UserForm/UserForm";
import CreateUserModal from "./components/CreateUserModal";
import EditUserModal from "./components/EditUserModal";

const baseClass = "user-management";

interface IAppSettingsPageProps {
  appConfigLoading: boolean;
  config: IConfig;
  currentUser: IUser;
  loadingTableData: boolean;
  invites: IInvite[];
  inviteErrors: { base: string; email: string;};
  isPremiumTier: boolean;
  users: IUser[];
  userErrors: { base: string; name: string;};
  teams: ITeam[];
}


// TODO: Try 1: define interface for formData and will get more helpful debugging
// TODO: Try 2: Consider re-writing this function all together....

const generateUpdateData = (currentUserData: any, formData: any) => {

  // array of updatable fields
  const updatableFields = [
    "global_role",
    "teams",
    "name",
    "email",
    "sso_enabled",
  ];

  // go over all the keys in the form data, reduce 
  return Object.keys(formData).reduce((updatedAttributes, attr) => {
    // attribute can be updated and is different from the current value.
    if (
      updatableFields.includes(attr) &&
      !isEqual(formData[attr], currentUserData[attr])
    ) {
      updatedAttributes[attr]: = formData[attr];
    }
    return updatedAttributes;
  }, {});
};

const UserManagementPage = ({
  appConfigLoading,
  config,
  currentUser,
  loadingTableData,
  invites,
  inviteErrors,
  isPremiumTier,
  users,
  userErrors,
  teams,
}: IAppSettingsPageProps): JSX.Element => {
  const dispatch = useDispatch();

  if (isPremiumTier) {
    // TODO: LOAD ALL TEAMS
  }

  // TODO: IMPLEMENT
    // Note: If the page is refreshed, `isPremiumTier` will be false at `componentDidMount` because
  // `config` will not have been loaded at that point. Accordingly, we need this lifecycle hook so
  // that `teams` information will be available to the edit user form.
  // componentDidUpdate(prevProps) {
  //   const { dispatch, isPremiumTier } = this.props;
  //   if (prevProps.isPremiumTier !== isPremiumTier) {
  //     isPremiumTier && dispatch(teamActions.loadAll({}));
  //   }
  // }

  // █▀ ▀█▀ ▄▀█ ▀█▀ █▀▀ █▀
  // ▄█ ░█░ █▀█ ░█░ ██▄ ▄█

  const [showCreateUserModal, setShowCreateUserModal] = useState<boolean>(
    false
  );
  const [showEditUserModal, setShowEditUserModal] = useState<boolean>(false);
  const [showDeleteUserModal, setShowDeleteUserModal] = useState<boolean>(
    false
  );
  const [showResetPasswordModal, setShowResetPasswordModal] = useState<boolean>(
    false
  );
  const [showResetSessionsModal, setShowResetSessionsModal] = useState<boolean>(
    false
  );
  const [isFormSubmitting, setIsFormSubmitting] = useState<boolean>(false);
  const [userEditing, setUserEditing] = useState<any>(null);
  const [usersEditing, setUsersEditing] = useState<any>([]);
  const [createUserErrors, setCreateUserErrors] = useState<any>({
    DEFAULT_CREATE_USER_ERRORS,
  });

  // ▀█▀ █▀█ █▀▀ █▀▀ █░░ █▀▀   █▀▄▀█ █▀█ █▀▄ ▄▀█ █░░ █▀
  // ░█░ █▄█ █▄█ █▄█ █▄▄ ██▄   █░▀░█ █▄█ █▄▀ █▀█ █▄▄ ▄█
    
 
    const toggleCreateUserModal = useCallback(() => {
      setShowCreateUserModal(!showCreateUserModal);
      
          // clear errors on close
    if (!showCreateUserModal) {
      setCreateUserErrors({ DEFAULT_CREATE_USER_ERRORS });
    }
  }, [showCreateUserModal, setShowCreateUserModal]);


    const toggleDeleteUserModal = useCallback(
    (user?: IUser) => {
        setShowDeleteUserModal(!showDeleteUserModal);
        // TODO: Decide which of these to use!
      user ? setUserEditing(user) : setUserEditing(undefined);
      setUserEditing(!showDeleteUserModal ? user : null);
    },
    [showDeleteUserModal, setShowDeleteUserModal, setUserEditing]
    );
  
    // added IInvite and undefined due to toggleeditusermodal being used later
    const toggleEditUserModal = useCallback(
    (user?: IUser) => {
        setShowEditUserModal(!showEditUserModal);
        // TODO: Decide which of these to use!
        user ? setUserEditing(user) : setUserEditing(undefined);
            setUserEditing(!showEditUserModal ? user : null);
    },
    [showEditUserModal, setShowEditUserModal, setUserEditing]
  );
  
  const toggleResetPasswordUserModal = useCallback(
    (user?: IUser) => {
    setShowResetPasswordModal(!showResetPasswordModal);
    setUserEditing(!showResetPasswordModal ? user : null);
    },
    [showResetPasswordModal, setShowResetPasswordModal, setUserEditing]
  );

  const toggleResetSessionsUserModal = useCallback(
    (user?: IUser) => {
    setShowResetSessionsModal(!showResetSessionsModal);
    setUserEditing(!showResetSessionsModal ? user : null);
    },
    [showResetSessionsModal, setShowResetSessionsModal, setUserEditing]
  );

  // █▀▀ █░█ █▄░█ █▀▀ ▀█▀ █ █▀█ █▄░█ █▀
  // █▀░ █▄█ █░▀█ █▄▄ ░█░ █ █▄█ █░▀█ ▄█
  
  const   getUser = (type: string, id: number) => {
    let userData;
    if (type === "user") {
      userData = users.find((user) => user.id === id);
    } else {
      userData = invites.find((invite) => invite.id === id);
    }
    return userData;
  };

  const onEditUser = (formData: any) => {
    const userData = getUser(userEditing.type, userEditing.id);

    const updatedAttrs = generateUpdateData(userData, formData);
    if (userEditing.type === "invite") {
      // Note: The edit invite action in this if block is occuring outside of Redux (unlike the
      // other cases below this block). Therefore, we must dispatch the loadAll action to ensure the
      // Redux store is updated.
      return Fleet.invites
        .update(userData, formData)
        .then(() => {
          dispatch(
            renderFlash("success", `Successfully edited ${userEditing?.name}`)
          );
          toggleEditUserModal();
        })
        .then(() => dispatch(inviteActions.loadAll()))
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

    if (userData?.email !== formData.email) {
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
            `Could not edit ${userEditing?.name}. Please try again.`
          )
        );
        toggleEditUserModal();
      });
  };

  const renderEditUserModal = () => {

    if (!showEditUserModal) return null;

    const userData = getUser(userEditing.type, userEditing.id);

    return (
      <Modal
        title="Edit user"
        onExit={toggleEditUserModal}
        className={`${baseClass}__edit-user-modal`}
      >
        <>
          <EditUserModal
            serverErrors={inviteErrors} // TODO: WTF, TYPES DON'T MATCH BETWEEN PAGES
            defaultEmail={userData?.email}
            defaultName={userData?.name}
            defaultGlobalRole={userData?.global_role}
            defaultTeams={userData?.teams}
            onCancel={toggleEditUserModal}
            onSubmit={onEditUser}
            availableTeams={teams}
            isPremiumTier={isPremiumTier}
            smtpConfigured={config.configured}
            canUseSso={config.enable_sso}
            isSsoEnabled={userData?.sso_enabled}
            isModifiedByGlobalAdmin
          />
        </>
      </Modal>
    );
  };

    const renderCreateUserModal = () => {
      const {
        currentUser,
        config,
        teams,
        userErrors,
        isPremiumTier,
      } = this.props;

      // TODO: REFACTOR TO TYPESCRIPT FOR THIS RENDER
      const { showCreateUserModal, isFormSubmitting } = this.state;
      const { onCreateUserSubmit, toggleCreateUserModal } = this;

      if (!showCreateUserModal) return null;

      return (
        <CreateUserModal
          serverErrors={userErrors}
          currentUserId={currentUser.id} // TODO: This is not used in CreateUserModal?!
          onCancel={toggleCreateUserModal}
          onSubmit={onCreateUserSubmit}
          availableTeams={teams}
          defaultGlobalRole={"observer"}
          defaultTeams={[]}
          defaultNewUserType={false}
          submitText={"Create"}
          isPremiumTier={isPremiumTier}
          smtpConfigured={config.configured}
          canUseSso={config.enable_sso}
          isFormSubmitting={isFormSubmitting}
          isModifiedByGlobalAdmin
          isNewUser
        />
      );
    };

  
    // TODO: START UP REFACTORING HERE FRIDAY
    const renderDeleteUserModal = () => {
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

    const renderResetPasswordModal = () => {
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

    const renderResetSessionsModal = () => {
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

  //   render() {
  //     const {
  //       renderCreateUserModal,
  //       renderEditUserModal,
  //       renderDeleteUserModal,
  //       renderResetPasswordModal,
  //       renderResetSessionsModal,
  //       toggleCreateUserModal,
  //       onTableQueryChange,
  //       onActionSelect,
  //     } = this;

  //     const {
  //       loadingTableData,
  //       users,
  //       invites,
  //       currentUser,
  //       isPremiumTier,
  //       userErrors,
  //     } = this.props;

  //     const tableHeaders = generateTableHeaders(onActionSelect, isPremiumTier);

  //     let tableData = [];
  //     if (!loadingTableData) {
  //       tableData = this.combineUsersAndInvites(
  //         users,
  //         invites,
  //         currentUser.id,
  //         onActionSelect
  //       );
  //     }

  return (
    <div className={`${baseClass} body-wrap`}>
      <p className={`${baseClass}__page-description`}>
        Create new users, customize user permissions, and remove users from
        Fleet.
      </p>
      {/* TODO: find a way to move these controls into the table component */}
      {users.length === 0 && Object.keys(userErrors).length > 0 ? (
        <TableDataError />
      ) : (
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
      )}
      {renderCreateUserModal()}
      {renderEditUserModal()}
      {renderDeleteUserModal()}
      {renderResetSessionsModal()}
      {renderResetPasswordModal()}
    </div>
  );
};

export default UserManagementPage;

// export class UserManagementPage extends Component {
//   static propTypes = {
//     appConfigLoading: PropTypes.bool,
//     config: configInterface,
//     currentUser: userInterface,
//     dispatch: PropTypes.func,
//     loadingTableData: PropTypes.bool,
//     invites: PropTypes.arrayOf(inviteInterface),
//     inviteErrors: PropTypes.shape({
//       base: PropTypes.string,
//       email: PropTypes.string,
//     }),
//     isPremiumTier: PropTypes.bool,
//     users: PropTypes.arrayOf(userInterface),
//     userErrors: PropTypes.shape({
//       base: PropTypes.string,
//       name: PropTypes.string,
//     }),
//     teams: PropTypes.arrayOf(teamInterface),
//   };

//   constructor(props) {
//     super(props);

//     this.state = {
//       showCreateUserModal: false,
//       showEditUserModal: false,
//       showDeleteUserModal: false,
//       showResetPasswordModal: false,
//       showResetSessionsModal: false,
//       isFormSubmitting: false,
//       userEditing: null,
//       usersEditing: [],
//       createUserErrors: { DEFAULT_CREATE_USER_ERRORS },
//     };
//   }

//   componentDidMount() {
//     const { dispatch, isPremiumTier } = this.props;
//     if (isPremiumTier) {
//       dispatch(teamActions.loadAll({}));
//     }
//   }

//   // Note: If the page is refreshed, `isPremiumTier` will be false at `componentDidMount` because
//   // `config` will not have been loaded at that point. Accordingly, we need this lifecycle hook so
//   // that `teams` information will be available to the edit user form.
//   componentDidUpdate(prevProps) {
//     const { dispatch, isPremiumTier } = this.props;
//     if (prevProps.isPremiumTier !== isPremiumTier) {
//       isPremiumTier && dispatch(teamActions.loadAll({}));
//     }
//   }

//   onCreateUserSubmit = (formData) => {
//     const { dispatch, config } = this.props;

//     this.setState({ isFormSubmitting: true });

//     if (formData.newUserType === NewUserType.AdminInvited) {
//       // Do some data formatting adding `invited_by` for the request to be correct and deleteing uncessary fields
//       const requestData = {
//         ...formData,
//         invited_by: formData.currentUserId,
//       };
//       delete requestData.currentUserId; // this field is not needed for the request
//       delete requestData.newUserType; // this field is not needed for the request
//       delete requestData.password; // this field is not needed for the request
//       dispatch(inviteActions.create(requestData))
//         .then(() => {
//           dispatch(
//             renderFlash(
//               "success",
//               `An invitation email was sent from ${config.sender_address} to ${formData.email}.`
//             )
//           );
//           this.toggleCreateUserModal();
//         })
//         .catch((userErrors) => {
//           if (userErrors.base?.includes("Duplicate")) {
//             dispatch(
//               renderFlash(
//                 "error",
//                 "A user with this email address already exists."
//               )
//             );
//           } else {
//             dispatch(
//               renderFlash("error", "Could not create user. Please try again.")
//             );
//           }
//         })
//         .finally(() => {
//           this.setState({ isFormSubmitting: false });
//         });
//     } else {
//       // Do some data formatting deleteing uncessary fields
//       const requestData = {
//         ...formData,
//       };
//       delete requestData.currentUserId; // this field is not needed for the request
//       delete requestData.newUserType; // this field is not needed for the request
//       dispatch(userActions.createUserWithoutInvitation(requestData))
//         .then(() => {
//           dispatch(
//             renderFlash("success", `Successfully created ${requestData.name}.`)
//           );
//           this.toggleCreateUserModal();
//         })
//         .catch((userErrors) => {
//           if (userErrors.base?.includes("Duplicate")) {
//             dispatch(
//               renderFlash(
//                 "error",
//                 "A user with this email address already exists."
//               )
//             );
//           } else {
//             dispatch(
//               renderFlash("error", "Could not create user. Please try again.")
//             );
//           }
//         })
//         .finally(() => {
//           this.setState({ isFormSubmitting: false });
//         });
//     }
//   };

//   onCreateCancel = (evt) => {
//     evt.preventDefault();
//     this.toggleCreateUserModal();
//   };

//   onDeleteUser = () => {
//     const { dispatch } = this.props;
//     const { userEditing } = this.state;
//     const { toggleDeleteUserModal } = this;

//     if (userEditing.type === "invite") {
//       dispatch(inviteActions.destroy(userEditing))
//         .then(() => {
//           dispatch(
//             renderFlash("success", `Successfully deleted ${userEditing?.name}.`)
//           );
//         })
//         .catch(() => {
//           dispatch(
//             renderFlash(
//               "error",
//               `Could not delete ${userEditing?.name}. Please try again.`
//             )
//           );
//         });
//       toggleDeleteUserModal();
//     } else {
//       dispatch(userActions.destroy(userEditing))
//         .then(() => {
//           dispatch(
//             renderFlash("success", `Successfully deleted ${userEditing?.name}.`)
//           );
//         })
//         .catch(() => {
//           dispatch(
//             renderFlash(
//               "error",
//               `Could not delete ${userEditing?.name}. Please try again.`
//             )
//           );
//         });
//       toggleDeleteUserModal();
//     }
//   };

//   onResetSessions = () => {
//     const { currentUser, dispatch } = this.props;
//     const { userEditing } = this.state;
//     const { toggleResetSessionsUserModal } = this;
//     const isResettingCurrentUser = currentUser.id === userEditing.id;

//     dispatch(userActions.deleteSessions(userEditing, isResettingCurrentUser))
//       .then(() => {
//         if (!isResettingCurrentUser) {
//           dispatch(renderFlash("success", "Sessions reset"));
//         }
//       })
//       .catch(() => {
//         dispatch(
//           renderFlash(
//             "error",
//             "Could not reset sessions for the selected user. Please try again."
//           )
//         );
//       });
//     toggleResetSessionsUserModal();
//   };

//   // NOTE: this is called once on the initial rendering. The initial render of
//   // the TableContainer child component calls this handler.
//   onTableQueryChange = (queryData) => {
//     const { dispatch } = this.props;
//     const {
//       pageIndex,
//       pageSize,
//       searchQuery,
//       sortHeader,
//       sortDirection,
//     } = queryData;
//     let sortBy = [];
//     if (sortHeader !== "") {
//       sortBy = [{ id: sortHeader, direction: sortDirection }];
//     }
//     dispatch(
//       userActions.loadAll({
//         page: pageIndex,
//         perPage: pageSize,
//         globalFilter: searchQuery,
//         sortBy,
//       })
//     );
//     dispatch(inviteActions.loadAll(pageIndex, pageSize, searchQuery, sortBy));
//   };

//   onActionSelect = (action, user) => {
//     const {
//       toggleEditUserModal,
//       toggleDeleteUserModal,
//       goToUserSettingsPage,
//       toggleResetPasswordUserModal,
//       toggleResetSessionsUserModal,
//     } = this;
//     switch (action) {
//       case "edit":
//         toggleEditUserModal(user);
//         break;
//       case "delete":
//         toggleDeleteUserModal(user);
//         break;
//       case "passwordReset":
//         toggleResetPasswordUserModal(user);
//         break;
//       case "resetSessions":
//         toggleResetSessionsUserModal(user);
//         break;
//       case "editMyAccount":
//         goToUserSettingsPage();
//         break;
//       default:
//         return null;
//     }
//     return null;
//   };

//   getUser = (type, id) => {
//     const { users, invites } = this.props;
//     let userData;
//     if (type === "user") {
//       userData = users.find((user) => user.id === id);
//     } else {
//       userData = invites.find((invite) => invite.id === id);
//     }
//     return userData;
//   };

//   combineUsersAndInvites = memoize((users, invites, currentUserId) => {
//     return combineDataSets(users, invites, currentUserId);
//   });

//   resetPassword = (user) => {
//     const { dispatch } = this.props;
//     const { toggleResetPasswordUserModal } = this;
//     const { requirePasswordReset } = userActions;

//     return dispatch(requirePasswordReset(user.id, { require: true })).then(
//       () => {
//         dispatch(
//           renderFlash(
//             "success",
//             "User required to reset password",
//             requirePasswordReset(user.id, { require: false }) // this is an undo action.
//           )
//         );
//         toggleResetPasswordUserModal();
//       }
//     );
//   };

//   goToUserSettingsPage = () => {
//     const { USER_SETTINGS } = paths;
//     const { dispatch } = this.props;

//     dispatch(push(USER_SETTINGS));
//   };

//   goToAppConfigPage = (evt) => {
//     evt.preventDefault();

//     const { ADMIN_SETTINGS } = paths;
//     const { dispatch } = this.props;

//     dispatch(push(ADMIN_SETTINGS));
//   };


//   render() {
//     const {
//       renderCreateUserModal,
//       renderEditUserModal,
//       renderDeleteUserModal,
//       renderResetPasswordModal,
//       renderResetSessionsModal,
//       toggleCreateUserModal,
//       onTableQueryChange,
//       onActionSelect,
//     } = this;

//     const {
//       loadingTableData,
//       users,
//       invites,
//       currentUser,
//       isPremiumTier,
//       userErrors,
//     } = this.props;

//     const tableHeaders = generateTableHeaders(onActionSelect, isPremiumTier);

//     let tableData = [];
//     if (!loadingTableData) {
//       tableData = this.combineUsersAndInvites(
//         users,
//         invites,
//         currentUser.id,
//         onActionSelect
//       );
//     }
//   }
// }

// const mapStateToProps = (state) => {
//   const stateEntityGetter = entityGetter(state);
//   const { config } = state.app;
//   const { loading: appConfigLoading } = state.app;
//   const { user: currentUser } = state.auth;
//   const { entities: users } = stateEntityGetter.get("users");
//   const { entities: invites } = stateEntityGetter.get("invites");
//   const { entities: teams } = stateEntityGetter.get("teams");
//   const {
//     errors: inviteErrors,
//     loading: loadingInvites,
//   } = state.entities.invites;
//   const { errors: userErrors, loading: loadingUsers } = state.entities.users;
//   const loadingTableData = loadingUsers || loadingInvites;
//   const isPremiumTier = permissionUtils.isPremiumTier(config);

//   return {
//     appConfigLoading,
//     config,
//     currentUser,
//     users,
//     userErrors,
//     invites,
//     inviteErrors,
//     isPremiumTier,
//     loadingTableData,
//     teams,
//   };
// };

// export default connect(mapStateToProps)(UserManagementPage);

