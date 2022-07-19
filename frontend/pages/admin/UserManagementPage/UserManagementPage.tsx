import React from "react";
import { InjectedRouter } from "react-router";
import SandboxGate from "components/SandboxGate";
import UsersTable from "./components/UsersTable";

const baseClass = "user-management";

interface IUserManagementProps {
  router: InjectedRouter; // v3
}

const UserManagementPage = ({ router }: IUserManagementProps): JSX.Element => {
  return (
    <div className={`${baseClass} body-wrap`}>
      <p className={`${baseClass}__page-description`}>
        Create new users, customize user permissions, and remove users from
        Fleet.
      </p>
      <SandboxGate
        message="User management is only available in self-managed Fleet"
        utmSource="fleet-ui-users-page"
      >
        <UsersTable router={router} />
      </SandboxGate>
    </div>
  );
};

export default UserManagementPage;
