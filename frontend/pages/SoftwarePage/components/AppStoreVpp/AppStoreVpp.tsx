import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { AxiosError } from "axios";

import PATHS from "router/paths";
import mdmAppleAPI, {
  IGetVppInfoResponse,
  IVppApp,
} from "services/entities/mdm_apple";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { buildQueryStringFromParams } from "utilities/url";
import { PLATFORM_DISPLAY_NAMES } from "interfaces/platform";
import { getErrorReason } from "interfaces/errors";
import { NotificationContext } from "context/notification";

import Card from "components/Card";
import CustomLink from "components/CustomLink";
import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import DataError from "components/DataError";
import Radio from "components/forms/fields/Radio";
import Checkbox from "components/forms/fields/Checkbox";

import SoftwareIcon from "../icons/SoftwareIcon";
import {
  generateRedirectQueryParams,
  getErrorMessage,
  getUniqueAppId,
} from "./helpers";

const baseClass = "app-store-vpp";

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
          text="Enable VPP"
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
        Add apps in{" "}
        <CustomLink url="https://business.apple.com" text="ABM" newTab /> Apps
        that are already added to this team are not listed.
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

interface IAppStoreVppProps {
  teamId: number;
  router: InjectedRouter;
  onExit: () => void;
  setAddedSoftwareToken: (token: string) => void;
}

const AppStoreVpp = ({
  teamId,
  router,
  onExit,
  setAddedSoftwareToken,
}: IAppStoreVppProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isSubmitDisabled, setIsSubmitDisabled] = useState(true);
  const [selectedApp, setSelectedApp] = useState<IVppApp | null>(null);
  const [isSelfService, setIsSelfService] = useState(false);

  const {
    data: vppInfo,
    isLoading: isLoadingVppInfo,
    error: errorVppInfo,
  } = useQuery<IGetVppInfoResponse, AxiosError>(
    ["vppInfo"],
    () => mdmAppleAPI.getVppInfo(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      staleTime: 30000,
      retry: (tries, error) => error.status !== 404 && tries <= 3,
    }
  );

  const {
    data: vppApps,
    isLoading: isLoadingVppApps,
    error: errorVppApps,
  } = useQuery(["vppSoftware", teamId], () => mdmAppleAPI.getVppApps(teamId), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: !!vppInfo,
    staleTime: 30000,
    select: (res) => res.app_store_apps,
  });

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
        teamId,
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

      const queryParams = generateRedirectQueryParams(teamId, isSelfService);
      // any unique string - triggers SW refetch
      setAddedSoftwareToken(`${Date.now()}`);
      router.push(`${PATHS.SOFTWARE}?${queryParams}`);
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    }
    onExit();
  };

  const renderContent = () => {
    if (isLoadingVppInfo || isLoadingVppApps) {
      return <Spinner />;
    }

    if (
      errorVppInfo &&
      getErrorReason(errorVppInfo).includes("MDMConfigAsset was not found")
    ) {
      return <EnableVppCard />;
    }

    if (errorVppInfo || errorVppApps) {
      return <DataError className={`${baseClass}__error`} />;
    }

    if (vppApps) {
      if (vppApps.length === 0) {
        return <NoVppAppsCard />;
      }
      return (
        <div className={`${baseClass}__modal-body`}>
          <VppAppList
            apps={vppApps}
            selectedApp={selectedApp}
            onSelect={onSelectApp}
          />
          <div className={`${baseClass}__help-text`}>
            These apps were added in Apple Business Manager (ABM). To add more
            apps, head to{" "}
            <CustomLink url="https://business.apple.com" text="ABM" newTab />
          </div>
          <Checkbox
            value={isSelfService}
            onChange={(newVal: boolean) => setIsSelfService(newVal)}
            className={`${baseClass}__self-service-checkbox`}
            tooltipContent={
              <>
                End users can install from <b>Fleet Desktop</b> {">"}{" "}
                <b>Self-service</b>.
              </>
            }
          >
            Self-service
          </Checkbox>
        </div>
      );
    }
    return null;
  };

  return (
    <div className={baseClass}>
      {renderContent()}
      <div className="modal-cta-wrap">
        <Button
          type="submit"
          variant="brand"
          disabled={isSubmitDisabled}
          onClick={onAddSoftware}
        >
          Add software
        </Button>
        <Button onClick={onExit} variant="inverse">
          Cancel
        </Button>
      </div>
    </div>
  );
};

export default AppStoreVpp;
