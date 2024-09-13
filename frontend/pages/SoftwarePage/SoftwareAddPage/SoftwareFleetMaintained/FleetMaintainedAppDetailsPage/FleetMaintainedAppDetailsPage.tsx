import React, { useContext, useState } from "react";
import { Location } from "history";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import softwareAPI from "services/entities/software";

import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import SidePanelContent from "components/SidePanelContent";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import { QueryContext } from "context/query";
import useToggleSidePanel from "hooks/useToggleSidePanel";
import { AppContext } from "context/app";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import AddPackage from "pages/SoftwarePage/components/AddPackage";
import AddPackageForm from "pages/SoftwarePage/components/AddPackageForm";
import AddPackageAdvancedOptions from "pages/SoftwarePage/components/AddPackageAdvancedOptions";

const baseClass = "fleet-maintained-app-details-page";

export interface IFleetMaintainedAppDetailsQueryParams {
  team_id?: string;
}

interface IFleetMaintainedAppDetailsRouteParams {
  id: string;
}

interface IFleetMaintainedAppDetailsPageProps {
  location: Location<IFleetMaintainedAppDetailsQueryParams>;
  routeParams: IFleetMaintainedAppDetailsRouteParams;
}

const FleetMaintainedAppDetailsPage = ({
  location,
  routeParams,
}: IFleetMaintainedAppDetailsPageProps) => {
  const teamId = location.query.team_id;
  const id = parseInt(routeParams.id, 10);

  const { isPremiumTier } = useContext(AppContext);
  const { selectedOsqueryTable, setSelectedOsqueryTable } = useContext(
    QueryContext
  );
  const { isSidePanelOpen, setSidePanelOpen } = useToggleSidePanel(true);

  const { data, isLoading, isError } = useQuery(
    ["fleet-maintained-app", id],
    () => softwareAPI.getFleetMainainedApp(id),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier,
      select: (res) => res.fleet_maintained_app,
    }
  );

  const onCloseSidebar = () => {
    setSidePanelOpen(false);
  };

  const onOpenSidebar = () => {
    setSidePanelOpen(true);
  };

  const onOsqueryTableSelect = (tableName: string) => {
    setSelectedOsqueryTable(tableName);
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
            path={`${
              PATHS.SOFTWARE_ADD_FLEET_MAINTAINED
            }?${buildQueryStringFromParams({ team_id: teamId })}`}
            className={`${baseClass}__back-to-add-software`}
          />
          <h1>{data.name}</h1>
          <AddPackageAdvancedOptions
            errors={{}}
            selectedPackage={IAddPackageFormData["software"]}
            preInstallQuery={data.pre_install_script}
            installScript={data.install_script}
            postInstallScript={data.post_install_script}
            uninstallScript={data.uninstall_script}
            onChangePreInstallQuery={(value?: string) => {}}
            onChangeInstallScript={(value: string) => {}}
            onChangePostInstallScript={(value?: string) => {}}
            onChangeUninstallScript={(value?: string) => {}}
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
            onClose={onCloseSidebar}
          />
        </SidePanelContent>
      )}
    </>
  );
};

export default FleetMaintainedAppDetailsPage;
