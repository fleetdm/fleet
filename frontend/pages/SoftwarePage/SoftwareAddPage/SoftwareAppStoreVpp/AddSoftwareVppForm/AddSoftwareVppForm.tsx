import React, { useContext, useState } from "react";

import PATHS from "router/paths";
import { PLATFORM_DISPLAY_NAMES } from "interfaces/platform";
import mdmAppleAPI, { IVppApp } from "services/entities/mdm_apple";
import { NotificationContext } from "context/notification";

import Card from "components/Card";
import CustomLink from "components/CustomLink";
import Radio from "components/forms/fields/Radio";
import Button from "components/buttons/Button";

import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import Checkbox from "components/forms/fields/Checkbox";
import { InjectedRouter } from "react-router";
import { buildQueryStringFromParams } from "utilities/url";
import { getErrorMessage, getUniqueAppId } from "./helpers";

const baseClass = "add-software-vpp-form";

const EnableVppCard = () => {
  return (
    <Card paddingSize="xxlarge" borderRadiusSize="medium">
      <div className={`${baseClass}__enable-vpp-message`}>
        <p className={`${baseClass}__enable-vpp-title`}>
          Volume Purchasing Program (VPP) isn&apos;t enabled
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
  <Card paddingSize="xxlarge" borderRadiusSize="medium">
    <div className={`${baseClass}__no-software-message`}>
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

interface IAddSoftwareVppFormProps {
  teamId: number;
  hasVppToken: boolean;
  router: InjectedRouter;
  vppApps?: IVppApp[];
}

const AddSoftwareVppForm = ({
  teamId,
  hasVppToken,
  router,
  vppApps,
}: IAddSoftwareVppFormProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isSubmitDisabled, setIsSubmitDisabled] = useState(true);
  const [selectedApp, setSelectedApp] = useState<IVppApp | null>(null);
  const [isSelfService, setIsSelfService] = useState(false);

  const goBackToSoftwareTitles = (availableInstall?: boolean) => {
    router.push(
      `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams({
        team_id: teamId,
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

      goBackToSoftwareTitles(true);
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    }
  };

  const renderContent = () => {
    if (!hasVppToken) {
      return <EnableVppCard />;
    }

    if (vppApps) {
      if (vppApps.length === 0) {
        return <NoVppAppsCard />;
      }

      return (
        <form>
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
        </form>
      );
    }

    return null;
  };

  return (
    <form className={baseClass}>
      <div className={`${baseClass}__content`}>{renderContent()}</div>
      <div className={`${baseClass}__action-buttons`}>
        <Button
          type="submit"
          variant="brand"
          disabled={isSubmitDisabled}
          onClick={onAddSoftware}
        >
          Add software
        </Button>
        <Button onClick={goBackToSoftwareTitles} variant="inverse">
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default AddSoftwareVppForm;
