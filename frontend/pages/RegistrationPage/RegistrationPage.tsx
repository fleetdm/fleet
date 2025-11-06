import React, { useContext, useState, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { max } from "lodash";

import paths from "router/paths";
import { AppContext } from "context/app";
import usersAPI from "services/entities/users";
import local from "utilities/local";

import FlashMessage from "components/FlashMessage";
import { INotification } from "interfaces/notification";

import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
// @ts-ignore
import RegistrationForm from "components/forms/RegistrationForm";
// @ts-ignore
import Breadcrumbs from "./Breadcrumbs";

const ERROR_NOTIFICATION: INotification = {
  alertType: "error",
  isVisible: true,
  message:
    "We were unable to configure Fleet. If your Fleet server is behind a proxy, please ensure the server can be reached.",
};

interface IRegistrationPageProps {
  router: InjectedRouter;
}

const baseClass = "registration-page";

const RegistrationPage = ({ router }: IRegistrationPageProps) => {
  const {
    currentUser,
    setCurrentUser,
    setAvailableTeams,
    setUserSettings,
  } = useContext(AppContext);
  const [page, setPage] = useState(1);
  const [pageProgress, setPageProgress] = useState(1);
  const [showSetupError, setShowSetupError] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    const { DASHBOARD } = paths;

    if (currentUser) {
      return router.push(DASHBOARD);
    }
  }, [currentUser]);

  const onNextPage = () => {
    const nextPage = page + 1;
    setPage(nextPage);
    setPageProgress(max([nextPage, pageProgress]) || 1);
  };

  const onRegistrationFormSubmit = async (formData: any) => {
    const { DASHBOARD } = paths;

    setIsLoading(true);
    try {
      const { token } = await usersAPI.setup(formData);
      local.setItem("auth_token", token);

      const { user, available_teams, settings } = await usersAPI.me();
      setCurrentUser(user);
      setAvailableTeams(user, available_teams);
      setUserSettings(settings);
      router.push(DASHBOARD);
      window.location.reload();
    } catch (error) {
      setIsLoading(false);
      setPage(1);
      setPageProgress(1);
      setShowSetupError(true);
    }
  };

  const onSetPage = (pageNum: number) => {
    if (pageNum > pageProgress) {
      return false;
    }

    setPage(pageNum);
  };

  const REGISTRATION_HEADERS: Record<number, string> = {
    1: "Set up user",
    2: "Organization details",
    3: "Set Fleet URL",
    4: "Confirm configuration",
  };
  const header = REGISTRATION_HEADERS[page];

  return (
    <AuthenticationFormWrapper
      className={baseClass}
      header={header}
      breadcrumbs={
        <Breadcrumbs
          currentPage={page}
          onSetPage={onSetPage}
          pageProgress={pageProgress}
        />
      }
    >
      <RegistrationForm
        page={page}
        onNextPage={onNextPage}
        onSubmit={onRegistrationFormSubmit}
        isLoading={isLoading}
      />
      {showSetupError && (
        <FlashMessage
          className={`${baseClass}__flash-message`}
          fullWidth={false}
          notification={ERROR_NOTIFICATION}
          onRemoveFlash={() => setShowSetupError(false)}
        />
      )}
    </AuthenticationFormWrapper>
  );
};

export default RegistrationPage;
