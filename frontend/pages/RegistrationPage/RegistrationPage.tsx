import React, { useState } from "react";
import { InjectedRouter } from "react-router";
import { max } from "lodash";

import paths from "router/paths"; // @ts-ignore
import usersAPI from "services/entities/users";

// @ts-ignore
import RegistrationForm from "components/forms/RegistrationForm"; // @ts-ignore
import Breadcrumbs from "./Breadcrumbs"; // @ts-ignore
import fleetLogoText from "../../../assets/images/fleet-logo-text-white.svg";

interface IRegistrationPageProps {
  router: InjectedRouter;
}

const RegistrationPage = ({ router }: IRegistrationPageProps) => {
  const [page, setPage] = useState<number>(1);
  const [pageProgress, setPageProgress] = useState<number>(1);

  const onNextPage = () => {
    const nextPage = page + 1;
    setPage(nextPage);
    setPageProgress(max([nextPage, pageProgress]) || 1);
  };

  const onRegistrationFormSubmit = async (formData: any) => {
    const { MANAGE_HOSTS } = paths;

    try {
      await usersAPI.setup(formData);
      return router.push(MANAGE_HOSTS);
    } catch (response) {
      console.error(response);
      return false;
    }
  };

  const onSetPage = (page: number) => {
    if (page > pageProgress) {
      return false;
    }

    setPage(page);
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
}

export default RegistrationPage;