import React, { useContext } from "react";
import { Location } from "history";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import softwareAPI from "services/entities/software";
import { QueryContext } from "context/query";
import { AppContext } from "context/app";
import useToggleSidePanel from "hooks/useToggleSidePanel";

import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import SidePanelContent from "components/SidePanelContent";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import AdvancedOptionsFields from "pages/SoftwarePage/components/AdvancedOptionsFields";
import FleetAppDetailsForm from "./FleetAppDetailsForm";
import { IFleetMaintainedAppFormData } from "./FleetAppDetailsForm/FleetAppDetailsForm";

const baseClass = "fleet-maintained-app-details-page";

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

const FleetMaintainedAppDetailsPage = ({
  location,
  router,
  routeParams,
}: IFleetMaintainedAppDetailsPageProps) => {
  const teamId = location.query.team_id;
  const id = parseInt(routeParams.id, 10);

  const { isPremiumTier } = useContext(AppContext);
  const { selectedOsqueryTable, setSelectedOsqueryTable } = useContext(
    QueryContext
  );
  const { isSidePanelOpen, setSidePanelOpen } = useToggleSidePanel(false);

  const { data, isLoading, isError } = useQuery(
    ["fleet-maintained-app", id],
    () => softwareAPI.getFleetMainainedApp(id),
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

  const onSubmit = (formData: IFleetMaintainedAppFormData) => {
    console.log(formData);
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

    if (data) {
      return (
        <>
          <BackLink
            text="Back to add software"
            path={backToAddSoftwareUrl}
            className={`${baseClass}__back-to-add-software`}
          />
          <h1>{data.name}</h1>
          <FleetAppDetailsForm
            showSchemaButton={isSidePanelOpen}
            defaultInstallScript={data.install_script}
            defaultPostInstallScript={data.post_install_script}
            defaultUninstallScript={data.uninstall_script}
            onClickShowSchema={() => setSidePanelOpen(true)}
            onCancel={onCancel}
            onSubmit={onSubmit}
          />
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
      {isPremiumTier && data && isSidePanelOpen && (
        <SidePanelContent className={`${baseClass}__side-panel`}>
          <QuerySidePanel
            key="query-side-panel"
            onOsqueryTableSelect={onOsqueryTableSelect}
            selectedOsqueryTable={selectedOsqueryTable}
            onClose={() => setSidePanelOpen(false)}
          />
        </SidePanelContent>
      )}
    </>
  );
};

export default FleetMaintainedAppDetailsPage;
