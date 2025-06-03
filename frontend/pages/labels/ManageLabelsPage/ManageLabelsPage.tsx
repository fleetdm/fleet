import React, { useContext, useCallback, useState } from "react";
import { InjectedRouter } from "react-router";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import Button from "components/buttons/Button";
import MainContent from "components/MainContent";
import TeamsDropdown from "components/TeamsDropdown";
import useTeamIdParam from "hooks/useTeamIdParam";
import PATHS from "router/paths";

const baseClass = "manage-labels-page";

interface IManageLabelsPageProps {
  router: InjectedRouter;
  location: {
    pathname: string;
    query: {
      team_id?: string;
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
    };
    search: string;
  };
}

const ManageLabelsPage = ({
  router,
  location,
}: IManageLabelsPageProps): JSX.Element => {
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
    isOnGlobalTeam,
    isPremiumTier,
    config,
  } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const onCreateLabelClick = useCallback(() => {
    router.push(PATHS.NEW_LABEL);
  }, [router]);

  // CTA button shows for all roles but global observers and current team's observers
  const canCreateLabel =
    isGlobalAdmin || isGlobalMaintainer || isTeamAdmin || isTeamMaintainer;

  const renderHeader = () => {
    if (isPremiumTier && userTeams && !config?.partnerships?.enable_primo) {
      if (userTeams.length > 1 || isOnGlobalTeam) {
        return (
          <TeamsDropdown
            currentUserTeams={userTeams}
            selectedTeamId={currentTeamId}
            onChange={onTeamChange}
          />
        );
      }
      if (userTeams.length === 1 && !isOnGlobalTeam) {
        return <h1>{userTeams[0].name}</h1>;
      }
    }
    return <h1>Labels</h1>;
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>{renderHeader()}</div>
            </div>
          </div>

          {canCreateLabel && (
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
          <p>Group hosts together to better organize your fleet.</p>
        </div>
        {/* Labels table will be added here in future implementation */}
        <div className={`${baseClass}__table-placeholder`}>
          <p>Labels table will be implemented here.</p>
        </div>
      </div>
    </MainContent>
  );
};

export default ManageLabelsPage;
