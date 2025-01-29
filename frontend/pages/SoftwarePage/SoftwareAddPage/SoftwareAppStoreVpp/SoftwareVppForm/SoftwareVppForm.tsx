import React, { useContext, useState } from "react";
import classnames from "classnames";

import PATHS from "router/paths";
import { ILabelSummary } from "interfaces/label";
import { PLATFORM_DISPLAY_NAMES } from "interfaces/platform";
import { IAppStoreApp } from "interfaces/software";
import mdmAppleAPI, { IVppApp } from "services/entities/mdm_apple";
import { NotificationContext } from "context/notification";

import Card from "components/Card";
import CustomLink from "components/CustomLink";
import Radio from "components/forms/fields/Radio";
import Button from "components/buttons/Button";
import FileUploader from "components/FileUploader";
import Checkbox from "components/forms/fields/Checkbox";
import TargetLabelSelector from "components/TargetLabelSelector";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";

import {
  generateFormValidation,
  getErrorMessage,
  getUniqueAppId,
} from "./helpers";
import { CUSTOM_TARGET_OPTIONS } from "../../SoftwareFleetMaintained/FleetMaintainedAppDetailsPage/FleetAppDetailsForm/helpers";

const baseClass = "software-vpp-form";

const EnableVppCard = () => {
  return (
    <Card borderRadiusSize="medium" paddingSize="xxxlarge">
      <div className={`${baseClass}__enable-vpp-message`}>
        <p className={`${baseClass}__enable-vpp-title`}>
          Volume Purchasing Program (VPP) isn&apos;t enabled
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

const EditVppCard = () => {
  return (
    <Card borderRadiusSize="medium" paddingSize="xxxlarge">
      <div className={`${baseClass}__enable-vpp-message`}>
        <p className={`${baseClass}__enable-vpp-title`}>
          This team isn&apos;t added to Volume Purchasing Program (VPP)
        </p>
        <p className={`${baseClass}__enable-vpp-description`}>
          To add App Store apps, first add this team to VPP.
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
  <Card borderRadiusSize="medium" paddingSize="xxxlarge">
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

export interface IVPPAppFormData {
  selfService: boolean;
  targetType: string;
  customTarget: string;
  labelTargets: Record<string, boolean>;
}

export interface IFormValidation {
  isValid: boolean;
  customTarget?: { isValid: boolean };
}

interface ISoftwareVppFormProps {
  labels: ILabelSummary[] | null;
  teamId: number;
  noVppTokenUploaded?: boolean;
  hasVppToken?: boolean;
  vppApps?: IVppApp[];
  softwareVppForEdit?: IAppStoreApp;
  onCancel: () => void;
}

const SoftwareVppForm = ({
  labels,
  teamId,
  noVppTokenUploaded,
  hasVppToken,
  vppApps,
  softwareVppForEdit,
  onCancel,
}: ISoftwareVppFormProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isSubmitDisabled, setIsSubmitDisabled] = useState(!softwareVppForEdit);
  const [selectedApp, setSelectedApp] = useState<IVppApp | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [formData, setFormData] = useState<IVPPAppFormData>({
    selfService: false,
    targetType: "All hosts",
    customTarget: "labelsIncludeAny",
    labelTargets: {},
  });
  const [formValidation, setFormValidation] = useState<IFormValidation>({
    isValid: true,
  });

  const onSelectApp = (app: IVppApp) => {
    setIsSubmitDisabled(false);
    setSelectedApp(app);
    if (app.platform === "ios" || app.platform === "ipados") {
      setFormData({ ...formData, selfService: false });
    }
  };

  const onAddSoftware = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    if (!selectedApp) {
      return;
    }

    setIsUploading(true);

    try {
      await mdmAppleAPI.addVppApp(
        teamId,
        selectedApp.app_store_id,
        selectedApp.platform,
        formData
      );
      renderFlash(
        "success",
        <>
          <b>{selectedApp.name}</b> successfully added.
        </>
      );

      onCancel; // Goes back to software titles
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    }

    setIsUploading(false);
  };

  const onSelectTargetType = (value: string) => {
    const newData = { ...formData, targetType: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onSelectCustomTargetOption = (value: string) => {
    const newData = { ...formData, customTarget: value };
    setFormData(newData);
  };

  const onSelectLabel = ({ name, value }: { name: string; value: boolean }) => {
    const newData = {
      ...formData,
      labelTargets: { ...formData.labelTargets, [name]: value },
    };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onEditSoftware = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    if (!selectedApp) {
      return;
    }

    setIsUploading(true);

    try {
      await mdmAppleAPI.addVppApp(
        teamId,
        selectedApp.app_store_id,
        selectedApp.platform,
        formData
      );
      renderFlash(
        "success",
        <>
          <b>{selectedApp.name}</b> successfully added.
        </>
      );

      onCancel; // Goes back to software titles
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    }

    setIsUploading(false);
  };

  const renderSelfServiceContent = (platform: string) => {
    if (platform !== "ios" && platform !== "ipados") {
      return (
        <Checkbox
          value={formData.selfService}
          onChange={(newVal: boolean) =>
            setFormData({ ...formData, selfService: newVal })
          }
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
      );
    }
    return null;
  };

  <SoftwareIcon name="appStore" size="medium" />;
  const renderContent = () => {
    if (softwareVppForEdit) {
      return (
        <div className={`${baseClass}__form-fields`}>
          <FileUploader
            canEdit={false}
            graphicName={"app-store"}
            message=".pkg, .msi, .exe, .deb, or .rpm"
            className={`${baseClass}__file-uploader`}
            fileDetails={
              softwareVppForEdit
                ? { name: softwareVppForEdit.name, platform: "macOS" }
                : undefined
            }
          />
          <TargetLabelSelector
            selectedTargetType={formData.targetType}
            selectedCustomTarget={formData.customTarget}
            selectedLabels={formData.labelTargets}
            customTargetOptions={CUSTOM_TARGET_OPTIONS}
            className={`${baseClass}__target`}
            onSelectTargetType={onSelectTargetType}
            onSelectCustomTarget={onSelectCustomTargetOption}
            onSelectLabel={onSelectLabel}
            labels={labels || []}
          />
          {renderSelfServiceContent(
            (selectedApp && selectedApp.platform) || ""
          )}
        </div>
      );
    }

    if (noVppTokenUploaded) {
      return <EnableVppCard />;
    }

    if (!hasVppToken) {
      return <EditVppCard />;
    }

    if (vppApps) {
      if (vppApps.length === 0) {
        return <NoVppAppsCard />;
      }

      return (
        <div className={`${baseClass}__form-fields`}>
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
          {renderSelfServiceContent(
            (selectedApp && selectedApp.platform) || ""
          )}
        </div>
      );
    }

    return null;
  };

  const contentWrapperClasses = classnames(`${baseClass}__content-wrapper`, {
    [`${baseClass}__content-disabled`]: isUploading,
  });

  return (
    <form
      className={baseClass}
      onSubmit={softwareVppForEdit ? onEditSoftware : onAddSoftware}
    >
      {isUploading && <div className={`${baseClass}__overlay`} />}
      <div className={contentWrapperClasses}>
        {!softwareVppForEdit && (
          <p>Apple App Store apps purchased via Apple Business Manager:</p>
        )}
        <div className={`${baseClass}__form-content`}>
          <>{renderContent()}</>
          <div className={`${baseClass}__action-buttons`}>
            <Button
              type="submit"
              variant="brand"
              disabled={isSubmitDisabled}
              isLoading={isUploading}
            >
              {softwareVppForEdit ? "Save" : "Add software"}
            </Button>
            <Button onClick={onCancel} variant="inverse">
              Cancel
            </Button>
          </div>
        </div>
      </div>
    </form>
  );
};

export default SoftwareVppForm;
