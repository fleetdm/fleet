import React from "react";
import { InjectedRouter } from "react-router";
import SecretsPaginatedList from "./components/SecretsPaginatedList/SecretsPaginatedList";

interface ISecretsProps {
  router: InjectedRouter; // v3
  teamIdForApi: number;
  currentPage: number;
}

const Secrets = ({ router, currentPage, teamIdForApi }: ISecretsProps) => {
  return (
    <>
      <p>
        Manage custom Secrets that will be available in scripts and profiles.
      </p>
      <SecretsPaginatedList />
    </>
  );
};

export default Secrets;
