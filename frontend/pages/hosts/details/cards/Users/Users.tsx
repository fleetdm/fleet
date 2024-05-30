import React from "react";

import { IHostUser } from "interfaces/host_users";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import Card from "components/Card";

import generateUsersTableHeaders from "./UsersTable/UsersTableConfig";

interface IUsersProps {
  users: IHostUser[];
  usersState: { username: string }[];
  isLoading: boolean;
  onUsersTableSearchChange: (queryData: ITableQueryData) => void;
  hostUsersEnabled?: boolean;
}

const baseClass = "users-card";

const Users = ({
  users,
  usersState,
  isLoading,
  onUsersTableSearchChange,
  hostUsersEnabled,
}: IUsersProps): JSX.Element => {
  const tableHeaders = generateUsersTableHeaders();

  if (!hostUsersEnabled) {
    return (
      <Card
        borderRadiusSize="large"
        includeShadow
        largePadding
        className={baseClass}
      >
        <p className="card__header">Users</p>
        <EmptyTable
          header="User collection has been disabled"
          info={
            <>
              Check out the Fleet documentation for{" "}
              <CustomLink
                url="https://fleetdm.com/docs/using-fleet/configuration-files#features"
                text="steps to enable this feature"
                newTab
              />
            </>
          }
        />
      </Card>
    );
  }

  return (
    <Card
      borderRadiusSize="large"
      includeShadow
      largePadding
      className={baseClass}
    >
      <>
        <p className="card__header">Users</p>
        {users?.length ? (
          <TableContainer
            columnConfigs={tableHeaders}
            data={usersState}
            isLoading={isLoading}
            defaultSortHeader="username"
            defaultSortDirection="asc"
            inputPlaceHolder="Search users by username"
            onQueryChange={onUsersTableSearchChange}
            resultsTitle="users"
            emptyComponent={() => (
              <EmptyTable
                header="No users match your search criteria"
                info="Try a different search."
              />
            )}
            showMarkAllPages={false}
            isAllPagesSelected={false}
            searchable
            wideSearch
            filteredCount={usersState.length}
            isClientSidePagination
          />
        ) : (
          <EmptyTable
            header="No users detected on this host"
            info="Expecting to see users? Try again in a few seconds as the system
              catches up."
          />
        )}
      </>
    </Card>
  );
};

export default Users;
