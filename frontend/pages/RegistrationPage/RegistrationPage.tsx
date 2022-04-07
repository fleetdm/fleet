import React, { useContext, useState, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { max } from "lodash";

import paths from "router/paths"; // @ts-ignore
import { AppContext } from "context/app";
import usersAPI from "services/entities/users";
import local from "utilities/local";

// @ts-ignore
import RegistrationForm from "components/forms/RegistrationForm"; // @ts-ignore
import Breadcrumbs from "./Breadcrumbs"; // @ts-ignore
import fleetLogoText from "../../../assets/images/fleet-logo-text-white.svg";

interface IRegistrationPageProps {
  router: InjectedRouter;
}

const RegistrationPage = ({ router }: IRegistrationPageProps) => {
  const { currentUser, setCurrentUser, setAvailableTeams } = useContext(
    AppContext
  );
  const [page, setPage] = useState<number>(1);
  const [pageProgress, setPageProgress] = useState<number>(1);

  useEffect(() => {
    const { HOME } = paths;

    if (currentUser) {
      return router.push(HOME);
    }
  }, [currentUser]);

  const onNextPage = () => {
    const nextPage = page + 1;
    setPage(nextPage);
    setPageProgress(max([nextPage, pageProgress]) || 1);
  };

  const onRegistrationFormSubmit = async (formData: any) => {
    const { MANAGE_HOSTS } = paths;

    try {
      const { token } = await usersAPI.setup(formData);
      local.setItem("auth_token", token);

      const { user, available_teams } = await usersAPI.me();
      setCurrentUser(user);
      setAvailableTeams(available_teams);
      return router.push(MANAGE_HOSTS);
    } catch (response) {
      console.error(response);
      return false;
    }
  };

  const onSetPage = (pageNum: number) => {
    if (pageNum > pageProgress) {
      return false;
    }

    setPage(pageNum);
  };

  return (
    <div className="registration-page">
      <img
        alt="Fleet logo"
        src={fleetLogoText}
        className="registration-page__logo"
      />
      <Breadcrumbs
        onClick={onSetPage}
        page={page}
        pageProgress={pageProgress}
      />
      <RegistrationForm
        page={page}
        onNextPage={onNextPage}
        onSubmit={onRegistrationFormSubmit}
      />
    </div>
  );
};

export default RegistrationPage;
