import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import PATHS from "router/paths";
import mdmAppleAPI, {
  IGetVppTokensResponse,
  IVppApp,
} from "services/entities/mdm_apple";
import { buildQueryStringFromParams } from "utilities/url";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  PLATFORM_DISPLAY_NAMES,
} from "utilities/constants";
import { NotificationContext } from "context/notification";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import Card from "components/Card";
import Radio from "components/forms/fields/Radio";

import AppStoreVpp from "pages/SoftwarePage/components/AppStoreVpp";
import {
  generateRedirectQueryParams,
  getErrorMessage,
  getUniqueAppId,
  teamHasVPPToken,
} from "pages/SoftwarePage/components/AppStoreVpp/helpers";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";

import { ISoftwareAddPageQueryParams } from "../SoftwareAddPage";
import AddSoftwareVppForm from "./AddSoftwareVppForm";

const baseClass = "software-app-store-vpp";

const EnableVppCard = () => {
  return (
    <Card borderRadiusSize="medium">
      <div className={`${baseClass}__enable-vpp`}>
        <p className={`${baseClass}__enable-vpp-title`}>
          <b>Volume Purchasing Program (VPP) isn&apos;t enabled</b>
        </p>
        <p className={`${baseClass}__enable-vpp-description`}>
          To add App Store apps, first enable VPP.
        </p>
        <CustomLink
          url={PATHS.ADMIN_INTEGRATIONS_VPP}
          text="Edit VPP"
          className={`${baseClass}__enable-vpp-link`}
        />
      </div>
    </Card>
  );
};

const NoVppAppsCard = () => (
  <Card borderRadiusSize="medium">
    <div className={`${baseClass}__no-software`}>
      <p className={`${baseClass}__no-software-title`}>
        You don&apos;t have any App Store apps
      </p>
      <p className={`${baseClass}__no-software-description`}>
        You must purchase apps in ABM. App Store apps that are already added to
        this team are not listed.
      </p>
    </div>
  </Card>
);

interface IVppAppListItemProps {
  app: IVppApp;
  selected: boolean;
  uniqueAppId: string;
  onSelect: (software: IVppApp) => void;
}

const VppAppListItem = ({
  app,
  selected,
  uniqueAppId,
  onSelect,
}: IVppAppListItemProps) => {
  return (
    <li className={`${baseClass}__list-item`}>
      <Radio
        label={
          <div className={`${baseClass}__app-info`}>
            <SoftwareIcon url={app.icon_url} />
            <span>{app.name}</span>
          </div>
        }
        id={`vppApp-${uniqueAppId}`}
        checked={selected}
        value={uniqueAppId}
        name="vppApp"
        onChange={() => onSelect(app)}
      />
      {app.platform && (
        <div className="app-platform">
          {PLATFORM_DISPLAY_NAMES[app.platform]}
        </div>
      )}
    </li>
  );
};

interface IVppAppListProps {
  apps: IVppApp[];
  selectedApp: IVppApp | null;
  onSelect: (app: IVppApp) => void;
}

const VppAppList = ({ apps, selectedApp, onSelect }: IVppAppListProps) => {
  const uniqueSelectedAppId = selectedApp ? getUniqueAppId(selectedApp) : null;
  return (
    <div className={`${baseClass}__list-container`}>
      <ul className={`${baseClass}__list`}>
        {apps.map((app) => {
          const uniqueAppId = getUniqueAppId(app);
          return (
            <VppAppListItem
              key={uniqueAppId}
              app={app}
              selected={uniqueSelectedAppId === uniqueAppId}
              uniqueAppId={uniqueAppId}
              onSelect={onSelect}
            />
          );
        })}
      </ul>
    </div>
  );
};

interface ISoftwareAppStoreProps {
  currentTeamId: number;
  router: InjectedRouter;
}

const SoftwareAppStoreVpp = ({
  currentTeamId,
  router,
}: ISoftwareAppStoreProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isSubmitDisabled, setIsSubmitDisabled] = useState(true);
  const [selectedApp, setSelectedApp] = useState<IVppApp | null>(null);
  const [isSelfService, setIsSelfService] = useState(false);

  const {
    data: vppInfo,
    isLoading: isLoadingVppInfo,
    error: errorVppInfo,
  } = useQuery<IGetVppTokensResponse, AxiosError>(
    ["vppInfo"],
    () => mdmAppleAPI.getVppTokens(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      staleTime: 30000,
      retry: (tries, error) => error.status !== 404 && tries <= 3,
    }
  );

  const hasVppToken = teamHasVPPToken(currentTeamId, vppInfo?.vpp_tokens);

  const {
    data: vppApps,
    isLoading: isLoadingVppApps,
    error: errorVppApps,
  } = useQuery(
    ["vppSoftware", currentTeamId],
    () => mdmAppleAPI.getVppApps(currentTeamId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: hasVppToken,
      staleTime: 30000,
      select: (res) => res.app_store_apps,
    }
  );

  const goBackToSoftwareTitles = (availableInstall?: boolean) => {
    router.push(
      `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams({
        team_id: currentTeamId,
        available_install: availableInstall,
      })}`
    );
  };

  const onSelectApp = (app: IVppApp) => {
    setIsSubmitDisabled(false);
    setSelectedApp(app);
  };

  const onAddSoftware = async () => {
    if (!selectedApp) {
      return;
    }

    try {
      await mdmAppleAPI.addVppApp(
        currentTeamId,
        selectedApp.app_store_id,
        selectedApp.platform,
        isSelfService
      );
      renderFlash(
        "success",
        <>
          <b>{selectedApp.name}</b> successfully added. Go to Host details page
          to install software.
        </>
      );

      goBackToSoftwareTitles(true);
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    }
  };

  const renderContent = () => {
    if (isLoadingVppInfo || isLoadingVppApps) {
      return <Spinner />;
    }

    if (errorVppInfo || errorVppApps) {
      return <DataError className={`${baseClass}__error`} />;
    }

    return (
      <div className={`${baseClass}__content`}>
        <p>Apple App Store apps purchased via Apple Business Manager:</p>
        <AddSoftwareVppForm
          router={router}
          teamId={currentTeamId}
          hasVppToken={hasVppToken}
          vppApps={vppApps}
        />
      </div>
    );
  };

  return <div className={baseClass}>{renderContent()}</div>;
};

export default SoftwareAppStoreVpp;
