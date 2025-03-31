import React, { useCallback } from "react";
import classnames from "classnames";

import { IHostUser } from "interfaces/host_users";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";
import EmptyTable from "components/EmptyTable";
import CustomLink from "components/CustomLink";
import Card from "components/Card";
import CardHeader from "components/CardHeader";

import generateTableHeaders from "./LocalUserAccountsTable/LocalUserAccountsTableConfig";

interface ILocalUserAccountsProps {
  users: IHostUser[];
  usersState: { username: string }[];
  isLoading: boolean;
  onUsersTableSearchChange: (queryData: ITableQueryData) => void;
  hostUsersEnabled?: boolean;
  className?: string;
}

const baseClass = "local-user-accounts-card";

const LocalUserAccounts = ({
  users,
  usersState,
  isLoading,
  onUsersTableSearchChange,
  hostUsersEnabled,
  className,
}: ILocalUserAccountsProps): JSX.Element => {
  const tableHeaders = generateTableHeaders();

  const renderUsersCount = useCallback(() => {
    return <TableCount name="items" count={usersState.length} />;
  }, [usersState.length]);

  if (!hostUsersEnabled) {
    return (
      <Card
        className={baseClass}
        borderRadiusSize="xxlarge"
        paddingSize="xlarge"
        includeShadow
      >
        <CardHeader header="Local user accounts" />
        <EmptyTable
          header="User collection has been disabled"
          info={
            <>
              Check out the Fleet documentation for{" "}
              <CustomLink
                url="https://fleetdm.com/learn-more-about/enable-user-collection"
                text="steps to enable this feature"
                newTab
              />
            </>
          }
        />
      </Card>
    );
  }

  const classNames = classnames(baseClass, className);

  return (
    <Card
      className={classNames}
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      includeShadow
    >
      <>
        <CardHeader header="Local user accounts" />
        {users?.length ? (
          <TableContainer
            columnConfigs={tableHeaders}
            data={usersState}
            isLoading={isLoading}
            defaultSortHeader="username"
            defaultSortDirection="asc"
            inputPlaceHolder="Search local user accounts by username"
            onQueryChange={onUsersTableSearchChange}
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
            renderCount={renderUsersCount}
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

export default LocalUserAccounts;
