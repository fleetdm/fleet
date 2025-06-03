import React, { useContext, useCallback } from "react";
import { InjectedRouter } from "react-router";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import Button from "components/buttons/Button";
import MainContent from "components/MainContent";
import PATHS from "router/paths";
import {
  isGlobalAdmin,
  isGlobalMaintainer,
  isOnGlobalTeam,
} from "utilities/permissions/permissions";

const baseClass = "manage-labels-page";

interface IManageLabelsPageProps {
  router: InjectedRouter;
  location: {
    pathname: string;
    query: {
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
    };
    search: string;
  };
}

const ManageLabelsPage = ({ router }: IManageLabelsPageProps): JSX.Element => {
  const { currentUser } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const onCreateLabelClick = useCallback(() => {
    router.push(PATHS.NEW_LABEL);
  }, [router]);

  if (!currentUser) {
    return <></>;
  }
  if (!isOnGlobalTeam(currentUser)) {
    // handling like this here since there is existing redirect logic at router level that needs to
    // be reconciled
    router.push("/404");
  }
  const canWriteLabels =
    isGlobalAdmin(currentUser) || isGlobalMaintainer(currentUser);

  const renderTable = () => {
    // TODO
    return <></>;
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>
                <h1>Labels</h1>
              </div>
            </div>
          </div>
          {canWriteLabels && (
            <div className={`${baseClass}__action-button-container`}>
              <Button
                className={`${baseClass}__create-button`}
                onClick={onCreateLabelClick}
              >
                Add label
              </Button>
            </div>
          )}
        </div>
        <div className={`${baseClass}__description`}>
          <p>Group hosts for targeting and filtering</p>
        </div>
        {renderTable()}
      </div>
    </MainContent>
  );
};

export default ManageLabelsPage;
