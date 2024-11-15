import React, { useContext, useState } from "react";
import { Location } from "history";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import softwareAPI from "services/entities/software";
import teamPoliciesAPI from "services/entities/team_policies";
import { QueryContext } from "context/query";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import { Platform, PLATFORM_DISPLAY_NAMES } from "interfaces/platform";
import useToggleSidePanel from "hooks/useToggleSidePanel";

import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import SidePanelContent from "components/SidePanelContent";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Card from "components/Card";

import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";

import FleetAppDetailsForm from "./FleetAppDetailsForm";
import { IFleetMaintainedAppFormData } from "./FleetAppDetailsForm/FleetAppDetailsForm";
import AddFleetAppSoftwareModal from "./AddFleetAppSoftwareModal";

import {
  getFleetAppPolicyDescription,
  getFleetAppPolicyName,
  getFleetAppPolicyQuery,
} from "./helpers";

const baseClass = "fleet-maintained-app-details-page";

interface ISoftwareSummaryProps {
  name: string;
  platform: string;
  version: string;
}

const FleetAppSummary = ({
  name,
  platform,
  version,
}: ISoftwareSummaryProps) => {
  return (
    <Card
      className={`${baseClass}__fleet-app-summary`}
      borderRadiusSize="medium"
    >
      <SoftwareIcon name={name} size="medium" />
      <div className={`${baseClass}__fleet-app-summary--details`}>
        <div className={`${baseClass}__fleet-app-summary--title`}>{name}</div>
        <div className={`${baseClass}__fleet-app-summary--info`}>
          <div className={`${baseClass}__fleet-app-summary--details--platform`}>
            {PLATFORM_DISPLAY_NAMES[platform as Platform]}
          </div>
          &bull;
          <div className={`${baseClass}__fleet-app-summary--details--version`}>
            {version}
          </div>
        </div>
      </div>
    </Card>
  );
};

export interface IFleetMaintainedAppDetailsQueryParams {
  team_id?: string;
}

interface IFleetMaintainedAppDetailsRouteParams {
  id: string;
}

interface IFleetMaintainedAppDetailsPageProps {
  location: Location<IFleetMaintainedAppDetailsQueryParams>;
  router: InjectedRouter;
  routeParams: IFleetMaintainedAppDetailsRouteParams;
}

/** This type includes the editable form data as well as the fleet maintained
 * app id */
export type IAddFleetMaintainedData = IFleetMaintainedAppFormData & {
  appId: number;
};

const FleetMaintainedAppDetailsPage = ({
  location,
  router,
  routeParams,
}: IFleetMaintainedAppDetailsPageProps) => {
  const teamId = location.query.team_id;
  const appId = parseInt(routeParams.id, 10);

  const { renderFlash } = useContext(NotificationContext);
  const { isPremiumTier } = useContext(AppContext);
  const { selectedOsqueryTable, setSelectedOsqueryTable } = useContext(
    QueryContext
  );
  const { isSidePanelOpen, setSidePanelOpen } = useToggleSidePanel(false);
  const [
    showAddFleetAppSoftwareModal,
    setShowAddFleetAppSoftwareModal,
  ] = useState(false);

  const { data: fleetApp, isLoading, isError } = useQuery(
    ["fleet-maintained-app", appId],
    () => softwareAPI.getFleetMainainedApp(appId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier,
      select: (res) => res.fleet_maintained_app,
    }
  );

  const onOsqueryTableSelect = (tableName: string) => {
    setSelectedOsqueryTable(tableName);
  };

  const backToAddSoftwareUrl = `${
    PATHS.SOFTWARE_ADD_FLEET_MAINTAINED
  }?${buildQueryStringFromParams({ team_id: teamId })}`;

  const onCancel = () => {
    router.push(backToAddSoftwareUrl);
  };

  const onSubmit = async (formData: IFleetMaintainedAppFormData) => {
    // this should not happen but we need to handle the type correctly
    if (!teamId) return;

    setShowAddFleetAppSoftwareModal(true);

    const { installType } = formData;
    try {
      await softwareAPI.addFleetMaintainedApp(parseInt(teamId, 10), {
        ...formData,
        appId,
      });

      if (installType === "manual") {
        router.push(
          `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams({
            team_id: teamId,
            available_for_install: true,
          })}`
        );
        renderFlash(
          "success",
          <>
            <b>{fleetApp?.name}</b> successfully added.
          </>
        );
      }
    } catch (error) {
      renderFlash("error", getErrorReason(error)); // TODO: handle error messages
    }

    if (installType === "automatic" && fleetApp) {
      try {
        await teamPoliciesAPI.create({
          name: getFleetAppPolicyName(fleetApp.name),
          description: getFleetAppPolicyDescription(fleetApp.name),
          query: getFleetAppPolicyQuery(fleetApp.id),
          team_id: parseInt(teamId, 10),
          platform: "darwin",
        });

        renderFlash(
          "success",
          <>
            <b>{fleetApp?.name}</b> successfully added.
          </>
        );
      } catch (e) {
        renderFlash(
          "error",
          "Couldn't add automatic install policy. Software is successfuly added. To try again delete software and add it again."
        );
      }

      // redirect will happen even on an error when we are creating a policy.
      router.push(
        `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams({
          team_id: teamId,
          available_for_install: true,
        })}`
      );
    }

    setShowAddFleetAppSoftwareModal(false);
  };

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }

    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError />;
    }

    if (fleetApp) {
      return (
        <>
          <BackLink
            text="Back to add software"
            path={backToAddSoftwareUrl}
            className={`${baseClass}__back-to-add-software`}
          />
          <h1>{fleetApp.name}</h1>
          <div className={`${baseClass}__page-content`}>
            <FleetAppSummary
              name={fleetApp.name}
              platform={fleetApp.platform}
              version={fleetApp.version}
            />
            <FleetAppDetailsForm
              showSchemaButton={!isSidePanelOpen}
              defaultInstallScript={fleetApp.install_script}
              defaultPostInstallScript={fleetApp.post_install_script}
              defaultUninstallScript={fleetApp.uninstall_script}
              onClickShowSchema={() => setSidePanelOpen(true)}
              onCancel={onCancel}
              onSubmit={onSubmit}
            />
          </div>
        </>
      );
    }

    return null;
  };

  return (
    <>
      <MainContent className={baseClass}>
        <>{renderContent()}</>
      </MainContent>
      {isPremiumTier && fleetApp && isSidePanelOpen && (
        <SidePanelContent className={`${baseClass}__side-panel`}>
          <QuerySidePanel
            key="query-side-panel"
            onOsqueryTableSelect={onOsqueryTableSelect}
            selectedOsqueryTable={selectedOsqueryTable}
            onClose={() => setSidePanelOpen(false)}
          />
        </SidePanelContent>
      )}
      {showAddFleetAppSoftwareModal && <AddFleetAppSoftwareModal />}
    </>
  );
};

export default FleetMaintainedAppDetailsPage;
