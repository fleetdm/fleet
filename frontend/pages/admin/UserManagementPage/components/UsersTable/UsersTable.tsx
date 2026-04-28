import React, { useState, useCallback, useContext, useMemo } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { IInvite } from "interfaces/invite";
import { IUser } from "interfaces/user";
import { IDropdownOption } from "interfaces/dropdownOption";
import authToken from "utilities/auth_token";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import usersAPI from "services/entities/users";
import invitesAPI from "services/entities/invites";

import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import TableDataError from "components/DataError";
import ActionsDropdown from "components/ActionsDropdown";
import EmptyState from "components/EmptyState";
import {
  generateTableHeaders,
  combineDataSets,
  IUserTableData,
} from "./UsersTableConfig";
import DeleteUserModal from "../DeleteUserModal";
import ResetPasswordModal from "../ResetPasswordModal";
import ResetSessionsModal from "../ResetSessionsModal";

const ADD_USER_OPTIONS: IDropdownOption[] = [
  {
    label: "Regular user",
    value: "human",
    helpText: "A human with access to Fleet",
  },
  {
    label: "API-only user",
    value: "api",
    helpText: "For GitOps or Fleet API automations",
  },
];

const EmptyUsersTable = () => (
  <EmptyState
    header="No users match the current criteria"
    info="Expecting to see users? Try again in a few seconds as the system catches up."
  />
);

interface IUsersTableProps {
  router: InjectedRouter; // v3
}
const UsersTable = ({ router }: IUsersTableProps): JSX.Element => {
  const { currentUser, isPremiumTier } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  // STATES
  const [showDeleteUserModal, setShowDeleteUserModal] = useState(false);
  const [showResetPasswordModal, setShowResetPasswordModal] = useState(false);
  const [showResetSessionsModal, setShowResetSessionsModal] = useState(false);
  const [isUpdatingUsers, setIsUpdatingUsers] = useState(false);
  const [userEditing, setUserEditing] = useState<IUserTableData | null>(null);
  const [querySearchText, setQuerySearchText] = useState("");

  // API CALLS
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

  // TOGGLE MODALS

  const toggleDeleteUserModal = useCallback(
    (user?: IUserTableData) => {
      setShowDeleteUserModal(!showDeleteUserModal);
      setUserEditing(!showDeleteUserModal ? user ?? null : null);
    },
    [showDeleteUserModal, setShowDeleteUserModal, setUserEditing]
  );

  const toggleResetPasswordUserModal = useCallback(
    (user?: IUserTableData) => {
      setShowResetPasswordModal(!showResetPasswordModal);
      setUserEditing(!showResetPasswordModal ? user ?? null : null);
    },
    [showResetPasswordModal, setShowResetPasswordModal, setUserEditing]
  );

  const toggleResetSessionsUserModal = useCallback(
    (user?: IUserTableData) => {
      setShowResetSessionsModal(!showResetSessionsModal);
      setUserEditing(!showResetSessionsModal ? user ?? null : null);
    },
    [showResetSessionsModal, setShowResetSessionsModal, setUserEditing]
  );

  // FUNCTIONS

  const onActionSelect = useCallback(
    (value: string, user: IUserTableData) => {
      switch (value) {
        case "edit": {
          const editPath = PATHS.ADMIN_USERS_EDIT(user.apiId);
          router.push(
            user.type === "invite" ? `${editPath}?type=invite` : editPath
          );
          break;
        }
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
          router.push(PATHS.ACCOUNT);
          break;
        default:
          return null;
      }
      return null;
    },
    [
      router,
      toggleDeleteUserModal,
      toggleResetPasswordUserModal,
      toggleResetSessionsUserModal,
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

  const onDeleteUser = () => {
    if (!userEditing) return;
    setIsUpdatingUsers(true);
    if (userEditing.type === "invite") {
      invitesAPI
        .destroy(userEditing.apiId)
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
        .destroy(userEditing.apiId)
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
    if (!userEditing) return;
    const isResettingCurrentUser =
      userEditing.type === "user" && currentUser?.id === userEditing.apiId;

    usersAPI
      .deleteSessions(userEditing.apiId)
      .then(() => {
        if (isResettingCurrentUser) {
          authToken.remove();
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

  const resetPassword = () => {
    if (!userEditing) return;
    usersAPI
      .requirePasswordReset(userEditing.apiId, { require: true })
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

  const renderDeleteUserModal = () => {
    if (!userEditing) return null;
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
        onResetConfirm={resetPassword}
        onResetCancel={toggleResetPasswordUserModal}
      />
    );
  };

  const renderResetSessionsModal = () => {
    return (
      <ResetSessionsModal
        onResetConfirm={onResetSessions}
        onResetCancel={toggleResetSessionsUserModal}
      />
    );
  };

  const tableHeaders = useMemo(
    () => generateTableHeaders(onActionSelect, isPremiumTier || false),
    [onActionSelect, isPremiumTier]
  );

  const loadingTableData = isFetchingUsers || isFetchingInvites;
  const tableDataError = loadingUsersError || loadingInvitesError;

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
    return <TableCount name="users" count={tableData?.length} />;
  }, [tableData?.length]);

  const onAddUserSelect = useCallback(
    (value: string) => {
      if (value === "human") {
        router.push(PATHS.ADMIN_USERS_NEW_HUMAN);
      } else if (value === "api") {
        router.push(PATHS.ADMIN_USERS_NEW_API);
      }
    },
    [router]
  );

  const renderAddUserControl = useCallback(
    () => (
      <ActionsDropdown
        options={ADD_USER_OPTIONS}
        onChange={onAddUserSelect}
        placeholder="Add user"
        variant="brand-button"
        buttonLabel="Add user"
        className="add-user-dropdown"
        menuAlign="left"
      />
    ),
    [onAddUserSelect]
  );

  return (
    <>
      {tableDataError ? (
        <TableDataError verticalPaddingSize="pad-xxxlarge" />
      ) : (
        <TableContainer
          columnConfigs={tableHeaders}
          data={tableData}
          isLoading={loadingTableData}
          defaultSortHeader="name"
          defaultSortDirection="asc"
          inputPlaceHolder="Search by name or email"
          customControl={renderAddUserControl}
          onQueryChange={onTableQueryChange}
          resultsTitle="users"
          emptyComponent={EmptyUsersTable}
          searchable
          showMarkAllPages={false}
          isAllPagesSelected={false}
          isClientSidePagination
          renderCount={renderUsersCount}
        />
      )}
      {showDeleteUserModal && renderDeleteUserModal()}
      {showResetSessionsModal && renderResetSessionsModal()}
      {showResetPasswordModal && renderResetPasswordModal()}
    </>
  );
};

export default UsersTable;
