import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import mdmAppleAPI, { IVppApp } from "services/entities/mdm_apple";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import Spinner from "components/Spinner";
import Button from "components/buttons/Button";
import DataError from "components/DataError";
import Radio from "components/forms/fields/Radio";
import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";
import { buildQueryStringFromParams } from "utilities/url";

const baseClass = "app-store-vpp";

interface IVppAppListItemProps {
  app: IVppApp;
  selected: boolean;
  onSelect: (software: IVppApp) => void;
}

const VppAppListItem = ({ app, selected, onSelect }: IVppAppListItemProps) => {
  return (
    <li className={`${baseClass}__list-item`}>
      <Radio
        label={app.name}
        id={`vppApp-${app.app_store_id}`}
        checked={selected}
        value={app.app_store_id.toString()}
        name="vppApp"
        onChange={() => onSelect(app)}
      />
    </li>
  );
};

interface IVppAppListProps {
  apps: IVppApp[];
  selectedApp: IVppApp | null;
  onSelect: (app: IVppApp) => void;
}

const VppAppList = ({ apps, selectedApp, onSelect }: IVppAppListProps) => {
  const renderContent = () => {
    if (apps.length === 0) {
      return (
        <div className={`${baseClass}__no-software`}>
          <p className={`${baseClass}__no-software-title`}>
            You don&apos;t have any App Store apps
          </p>
          <p className={`${baseClass}__no-software-description`}>
            You must purchase apps in ABM. App Store apps that are already added
            to this team are not listed.
          </p>
        </div>
      );
    }

    return (
      <ul className={`${baseClass}__list`}>
        {apps.map((app) => (
          <VppAppListItem
            key={app.app_store_id}
            app={app}
            selected={selectedApp?.app_store_id === app.app_store_id}
            onSelect={onSelect}
          />
        ))}
      </ul>
    );
  };

  return (
    <div className={`${baseClass}__list-container`}>{renderContent()}</div>
  );
};

interface IAppStoreVppProps {
  teamId: number;
  router: InjectedRouter;
  onExit: () => void;
}

const AppStoreVpp = ({ teamId, router, onExit }: IAppStoreVppProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isSubmitDisabled, setIsSubmitDisabled] = useState(true);
  const [selectedApp, setSelectedApp] = useState<IVppApp | null>(null);

  const { data: vppApps, isLoading, isError } = useQuery(
    "vppSoftware",
    () => mdmAppleAPI.getVppApps(teamId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (res) => res.app_store_apps,
    }
  );

  const onSelectApp = (app: IVppApp) => {
    setIsSubmitDisabled(false);
    setSelectedApp(app);
  };

  const onAddSoftware = async () => {
    if (!selectedApp) {
      return;
    }

    try {
      await mdmAppleAPI.addVppApp(teamId, selectedApp.app_store_id);
      renderFlash(
        "success",
        <>
          <b>{selectedApp.name}</b> successfully added. Go to Host details page
          to install software.
        </>
      );
      const queryParams = buildQueryStringFromParams({
        team_id: teamId,
        available_for_install: true,
      });
      router.push(`${PATHS.SOFTWARE}?${queryParams}`);
    } catch (e) {
      renderFlash("error", getErrorReason(e));
    }
    onExit();
  };

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError className={`${baseClass}__error`} />;
    }

    return vppApps ? (
      <VppAppList
        apps={vppApps}
        selectedApp={selectedApp}
        onSelect={onSelectApp}
      />
    ) : null;
  };

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Apple App Store apps purchased via Apple Business Manager.
      </p>
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
