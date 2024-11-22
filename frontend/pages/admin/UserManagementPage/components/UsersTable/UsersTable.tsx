import React, { useState, useCallback, useContext, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import paths from "router/paths";
import { IApiError } from "interfaces/errors";
import { IInvite, IEditInviteFormData } from "interfaces/invite";
import { IUser, IUserFormErrors } from "interfaces/user";
import { ITeam } from "interfaces/team";
import { clearToken } from "utilities/local";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import teamsAPI, { ILoadTeamsResponse } from "services/entities/teams";
import usersAPI from "services/entities/users";
import invitesAPI from "services/entities/invites";

import { DEFAULT_CREATE_USER_ERRORS as DEFAULT_ADD_USER_ERRORS } from "utilities/constants";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import TableDataError from "components/DataError";
import EmptyTable from "components/EmptyTable";
import { generateTableHeaders, combineDataSets } from "./UsersTableConfig";
import DeleteUserModal from "../DeleteUserModal";
import ResetPasswordModal from "../ResetPasswordModal";
import ResetSessionsModal from "../ResetSessionsModal";
import { NewUserType, IFormData } from "../UserForm/UserForm";
import AddUserModal from "../AddUserModal";
import EditUserModal from "../EditUserModal";

const EmptyUsersTable = () => (
  <EmptyTable
    header="No users match the current criteria"
    info="Expecting to see users? Try again in a few seconds as the system catches up."
  />
);

interface IUsersTableProps {
  router: InjectedRouter; // v3
}

const UsersTable = ({ router }: IUsersTableProps): JSX.Element => {
  const { config, currentUser, isPremiumTier } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  // STATES
  const [showAddUserModal, setShowAddUserModal] = useState(false);
  const [showEditUserModal, setShowEditUserModal] = useState(false);
  const [showDeleteUserModal, setShowDeleteUserModal] = useState(false);
  const [showResetPasswordModal, setShowResetPasswordModal] = useState(false);
  const [showResetSessionsModal, setShowResetSessionsModal] = useState(false);
  const [isUpdatingUsers, setIsUpdatingUsers] = useState(false);
  const [userEditing, setUserEditing] = useState<any>(null);
  const [addUserErrors, setAddUserErrors] = useState<IUserFormErrors>(
    DEFAULT_ADD_USER_ERRORS
  );
  const [editUserErrors, setEditUserErrors] = useState<IUserFormErrors>(
    DEFAULT_ADD_USER_ERRORS
  );
  const [querySearchText, setQuerySearchText] = useState("");

  // API CALLS
  const {
    data: teams,
    isFetching: isFetchingTeams,
    error: loadingTeamsError,
  } = useQuery<ILoadTeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      enabled: !!isPremiumTier,
      select: (data: ILoadTeamsResponse) => data.teams,
    }
  );

  const {
    data: users,
    isFetching: isFetchingUsers,
    error: loadingUsersError,
    refetch: refetchUsers,
  } = useQuery<IUser[], Error, IUser[]>(
    ["users", querySearchText],
    () => usersAPI.loadAll({ globalFilter: querySearchText }),
    {
      select: (data: IUser[]) => data,
    }
  );

  const {
    data: invites,
    isFetching: isFetchingInvites,
    error: loadingInvitesError,
    refetch: refetchInvites,
  } = useQuery<IInvite[], Error, IInvite[]>(
    ["invites", querySearchText],
    () => invitesAPI.loadAll({ globalFilter: querySearchText }),
    {
      select: (data: IInvite[]) => {
        return data;
      },
    }
  );

  // TODO: Cleanup useCallbacks, add missing dependencies, use state setter functions, e.g.,
  // `setShowAddUserModal((prevState) => !prevState)`, instead of including state
  // variables as dependencies for toggles, etc.

  // TOGGLE MODALS

  const toggleAddUserModal = useCallback(() => {
    setShowAddUserModal(!showAddUserModal);

    // clear errors on close
    if (!showAddUserModal) {
      setAddUserErrors(DEFAULT_ADD_USER_ERRORS);
    }
  }, [showAddUserModal, setShowAddUserModal]);

  const toggleDeleteUserModal = useCallback(
    (user?: IUser | IInvite) => {
      setShowDeleteUserModal(!showDeleteUserModal);
      setUserEditing(!showDeleteUserModal ? user : null);
    },
    [showDeleteUserModal, setShowDeleteUserModal, setUserEditing]
  );

  const toggleEditUserModal = useCallback(
    (user?: IUser | IInvite) => {
      setShowEditUserModal(!showEditUserModal);
      setUserEditing(!showEditUserModal ? user : null);
      setEditUserErrors(DEFAULT_ADD_USER_ERRORS);
    },
    [showEditUserModal, setShowEditUserModal, setUserEditing]
  );

  const toggleResetPasswordUserModal = useCallback(
    (user?: IUser | IInvite) => {
      setShowResetPasswordModal(!showResetPasswordModal);
      setUserEditing(!showResetPasswordModal ? user : null);
    },
    [showResetPasswordModal, setShowResetPasswordModal, setUserEditing]
  );

  const toggleResetSessionsUserModal = useCallback(
    (user?: IUser | IInvite) => {
      setShowResetSessionsModal(!showResetSessionsModal);
      setUserEditing(!showResetSessionsModal ? user : null);
    },
    [showResetSessionsModal, setShowResetSessionsModal, setUserEditing]
  );

  // FUNCTIONS

  const goToAccountPage = useCallback(() => {
    const { ACCOUNT } = paths;
    router.push(ACCOUNT);
  }, [router]);

  const onActionSelect = useCallback(
    (value: string, user: IUser | IInvite) => {
      switch (value) {
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
          goToAccountPage();
          break;
        default:
          return null;
      }
      return null;
    },
    [
      toggleEditUserModal,
      toggleDeleteUserModal,
      toggleResetPasswordUserModal,
      toggleResetSessionsUserModal,
      goToAccountPage,
    ]
  );

  const onTableQueryChange = useCallback(
    (queryData: ITableQueryData) => {
      const { searchQuery } = queryData;

      setQuerySearchText(searchQuery);

      refetchUsers();
      refetchInvites();
    },
    [refetchUsers, refetchInvites]
  );

  const getUser = (type: string, id: number) => {
    let userData;
    if (type === "user") {
      userData = users?.find((user) => user.id === id);
    } else {
      userData = invites?.find((invite) => invite.id === id);
    }
    return userData;
  };

  const onAddUserSubmit = (formData: IFormData) => {
    setIsUpdatingUsers(true);

    if (formData.newUserType === NewUserType.AdminInvited) {
      // Do some data formatting adding `invited_by` for the request to be correct and deleteing uncessary fields
      const requestData = {
        ...formData,
        invited_by: formData.currentUserId,
      };
      delete requestData.currentUserId; // this field is not needed for the request
      delete requestData.newUserType; // this field is not needed for the request
      delete requestData.password; // this field is not needed for the request
      invitesAPI
        .create(requestData)
        .then(() => {
          const senderAddressMessage = config?.smtp_settings?.sender_address
            ? ` from ${config?.smtp_settings?.sender_address}`
            : "";
          renderFlash(
            "success",
            `An invitation email was sent${senderAddressMessage} to ${formData.email}.`
          );
          toggleAddUserModal();
          refetchInvites();
        })
        .catch((userErrors: { data: IApiError }) => {
          if (userErrors.data.errors[0].reason.includes("already exists")) {
            setAddUserErrors({
              email: "A user with this email address already exists",
            });
          } else if (
            userErrors.data.errors[0].reason.includes("required criteria")
          ) {
            setAddUserErrors({
              password: "Password must meet the criteria below",
            });
          } else if (
            userErrors.data.errors?.[0].reason.includes("password too long")
          ) {
            setAddUserErrors({
              password: "Password is over the character limit.",
            });
          } else {
            renderFlash("error", "Could not create user. Please try again.");
          }
        })
        .finally(() => {
          setIsUpdatingUsers(false);
        });
    } else {
      // Do some data formatting deleting unnecessary fields
      const requestData = {
        ...formData,
      };
      delete requestData.currentUserId; // this field is not needed for the request
      delete requestData.newUserType; // this field is not needed for the request
      usersAPI
        .createUserWithoutInvitation(requestData)
        .then(() => {
          renderFlash("success", `Successfully created ${requestData.name}.`);
          toggleAddUserModal();
          refetchUsers();
        })
        .catch((userErrors: { data: IApiError }) => {
          if (userErrors.data.errors[0].reason.includes("Duplicate")) {
            setAddUserErrors({
              email: "A user with this email address already exists",
            });
          } else if (
            userErrors.data.errors[0].reason.includes("required criteria")
          ) {
            setAddUserErrors({
              password: "Password must meet the criteria below",
            });
          } else if (
            userErrors.data.errors?.[0].reason.includes("password too long")
          ) {
            setAddUserErrors({
              password: "Password is over the character limit.",
            });
          } else {
            renderFlash("error", "Could not create user. Please try again.");
          }
        })
        .finally(() => {
          setIsUpdatingUsers(false);
        });
    }
  };

  const onEditUser = (formData: IFormData) => {
    const userData = getUser(userEditing.type, userEditing.id);

    let userUpdatedFlashMessage = `Successfully edited ${formData.name}`;
    if (userData?.email !== formData.email) {
      const senderAddressMessage = config?.smtp_settings?.sender_address
        ? ` from ${config?.smtp_settings?.sender_address}`
        : "";
      userUpdatedFlashMessage += `: A confirmation email was sent${senderAddressMessage} to ${formData.email}`;
    }
    const userUpdatedEmailError =
      "A user with this email address already exists";
    const userUpdatedPasswordError = "Password must meet the criteria below";
    const userUpdatedError = `Could not edit ${userEditing?.name}. Please try again.`;

    // Do not update password to empty string
    const requestData = formData;
    if (requestData.new_password === "") {
      requestData.new_password = null;
    }

    setIsUpdatingUsers(true);
    if (userEditing.type === "invite") {
      return (
        userData &&
        invitesAPI
          .update(userData.id, requestData as IEditInviteFormData)
          .then(() => {
            renderFlash("success", userUpdatedFlashMessage);
            toggleEditUserModal();
            refetchInvites();
          })
          .catch((userErrors: { data: IApiError }) => {
            if (userErrors.data.errors[0].reason.includes("already exists")) {
              setEditUserErrors({
                email: userUpdatedEmailError,
              });
            } else if (
              userErrors.data.errors[0].reason.includes("required criteria")
            ) {
              setEditUserErrors({
                password: userUpdatedPasswordError,
              });
            } else {
              renderFlash("error", userUpdatedError);
            }
          })
          .finally(() => {
            setIsUpdatingUsers(false);
          })
      );
    }

    return (
      userData &&
      usersAPI
        .update(userData.id, requestData)
        .then(() => {
          renderFlash("success", userUpdatedFlashMessage);
          toggleEditUserModal();
          refetchUsers();
        })
        .catch((userErrors: { data: IApiError }) => {
          if (userErrors.data.errors[0].reason.includes("already exists")) {
            setEditUserErrors({
              email: userUpdatedEmailError,
            });
          } else if (
            userErrors.data.errors[0].reason.includes("required criteria")
          ) {
            setEditUserErrors({
              password: userUpdatedPasswordError,
            });
          } else {
            renderFlash("error", userUpdatedError);
          }
        })
        .finally(() => {
          setIsUpdatingUsers(false);
        })
    );
  };

  const onDeleteUser = () => {
    setIsUpdatingUsers(true);
    if (userEditing.type === "invite") {
      invitesAPI
        .destroy(userEditing.id)
        .then(() => {
          renderFlash("success", `Successfully deleted ${userEditing?.name}.`);
        })
        .catch(() => {
          renderFlash(
            "error",
            `Could not delete ${userEditing?.name}. Please try again.`
          );
        })
        .finally(() => {
          toggleDeleteUserModal();
          refetchInvites();
          setIsUpdatingUsers(false);
        });
    } else {
      usersAPI
        .destroy(userEditing.id)
        .then(() => {
          renderFlash("success", `Successfully deleted ${userEditing?.name}.`);
        })
        .catch(() => {
          renderFlash(
            "error",
            `Could not delete ${userEditing?.name}. Please try again.`
          );
        })
        .finally(() => {
          toggleDeleteUserModal();
          refetchUsers();
          setIsUpdatingUsers(false);
        });
    }
  };

  const onResetSessions = () => {
    const isResettingCurrentUser = currentUser?.id === userEditing.id;

    usersAPI
      .deleteSessions(userEditing.id)
      .then(() => {
        if (isResettingCurrentUser) {
          clearToken();
          setTimeout(() => {
            window.location.href = "/";
          }, 500);
          return;
        }
        renderFlash("success", "Successfully reset sessions.");
      })
      .catch(() => {
        renderFlash("error", "Could not reset sessions. Please try again.");
      })
      .finally(() => {
        toggleResetSessionsUserModal();
      });
  };

  const resetPassword = (user: IUser) => {
    return usersAPI
      .requirePasswordReset(user.id, { require: true })
      .then(() => {
        renderFlash("success", "Successfully required a password reset.");
      })
      .catch(() => {
        renderFlash(
          "error",
          "Could not require a password reset. Please try again."
        );
      })
      .finally(() => {
        toggleResetPasswordUserModal();
      });
  };

  const renderEditUserModal = () => {
    const userData = getUser(userEditing.type, userEditing.id);

    return (
      <EditUserModal
        defaultEmail={userData?.email}
        defaultName={userData?.name}
        defaultGlobalRole={userData?.global_role}
        defaultTeams={userData?.teams}
        onCancel={toggleEditUserModal}
        onSubmit={onEditUser}
        availableTeams={teams || []}
        isPremiumTier={isPremiumTier || false}
        smtpConfigured={config?.smtp_settings?.configured || false}
        sesConfigured={config?.email?.backend === "ses" || false}
        canUseSso={config?.sso_settings.enable_sso || false}
        isSsoEnabled={userData?.sso_enabled}
        isTwoFactorAuthenticationEnabled={
          userData?.two_factor_authentication_enabled
        }
        isApiOnly={userData?.api_only || false}
        isModifiedByGlobalAdmin
        isInvitePending={userEditing.type === "invite"}
        editUserErrors={editUserErrors}
        isUpdatingUsers={isUpdatingUsers}
      />
    );
  };

  const renderAddUserModal = () => {
    return (
      <AddUserModal
        addUserErrors={addUserErrors}
        onCancel={toggleAddUserModal}
        onSubmit={onAddUserSubmit}
        availableTeams={teams || []}
        defaultGlobalRole="observer"
        defaultTeams={[]}
        isPremiumTier={isPremiumTier || false}
        smtpConfigured={config?.smtp_settings?.configured || false}
        sesConfigured={config?.email?.backend === "ses" || false}
        canUseSso={config?.sso_settings.enable_sso || false}
        isUpdatingUsers={isUpdatingUsers}
        isModifiedByGlobalAdmin
      />
    );
  };

  const renderDeleteUserModal = () => {
    return (
      <DeleteUserModal
        name={userEditing.name}
        onDelete={onDeleteUser}
        onCancel={toggleDeleteUserModal}
        isUpdatingUsers={isUpdatingUsers}
      />
    );
  };

  const renderResetPasswordModal = () => {
    return (
      <ResetPasswordModal
        user={userEditing}
        onResetConfirm={resetPassword}
        onResetCancel={toggleResetPasswordUserModal}
      />
    );
  };

  const renderResetSessionsModal = () => {
    return (
      <ResetSessionsModal
        user={userEditing}
        onResetConfirm={onResetSessions}
        onResetCancel={toggleResetSessionsUserModal}
      />
    );
  };

  const tableHeaders = useMemo(
    () => generateTableHeaders(onActionSelect, isPremiumTier || false),
    [onActionSelect, isPremiumTier]
  );

  const loadingTableData =
    isFetchingUsers || isFetchingInvites || isFetchingTeams;
  const tableDataError =
    loadingUsersError || loadingInvitesError || loadingTeamsError;

  const tableData = useMemo(
    () =>
      !loadingTableData &&
      !tableDataError &&
      users &&
      invites &&
      currentUser?.id
        ? combineDataSets(users, invites, currentUser.id)
        : [],
    [loadingTableData, tableDataError, users, invites, currentUser?.id]
  );

  const renderUsersCount = useCallback(() => {
    return <TableCount name="users" count={users?.length} />;
  }, [users?.length]);

  return (
    <>
      {tableDataError ? (
        <TableDataError />
      ) : (
        <TableContainer
          columnConfigs={tableHeaders}
          data={tableData}
          isLoading={loadingTableData}
          defaultSortHeader="name"
          defaultSortDirection="asc"
          inputPlaceHolder="Search by name or email"
          actionButton={{
            name: "add user",
            buttonText: "Add user",
            onActionButtonClick: toggleAddUserModal,
          }}
          onQueryChange={onTableQueryChange}
          resultsTitle={"users"}
          emptyComponent={EmptyUsersTable}
          searchable
          showMarkAllPages={false}
          isAllPagesSelected={false}
          isClientSidePagination
          renderCount={renderUsersCount}
        />
      )}
      {showAddUserModal && renderAddUserModal()}
      {showEditUserModal && renderEditUserModal()}
      {showDeleteUserModal && renderDeleteUserModal()}
      {showResetSessionsModal && renderResetSessionsModal()}
      {showResetPasswordModal && renderResetPasswordModal()}
    </>
  );
};

export default UsersTable;
