import React from "react";
import { InjectedRouter } from "react-router";
import UsersTable from "./components/UsersTable";

const baseClass = "user-management";

interface IUserManagementProps {
  router: InjectedRouter; // v3
}

const UserManagementPage = ({ router }: IUserManagementProps): JSX.Element => {
  return (
    <div className={`${baseClass}`}>
      <p className={`${baseClass}__page-description`}>
        Create new users, customize user permissions, and remove users from
        Fleet.
      </p>
      <UsersTable router={router} />
    </div>
  );
};

export default UserManagementPage;
