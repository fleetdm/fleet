import React, { useContext, useState } from "react";
import classnames from "classnames";

import { AppContext } from "context/app";

import { ILabelSummary } from "interfaces/label";
import { PLATFORM_DISPLAY_NAMES } from "interfaces/platform";
import { IAppStoreApp } from "interfaces/software";
import { IVppApp } from "services/entities/mdm_apple";

import CustomLink from "components/CustomLink";
import Radio from "components/forms/fields/Radio";
import Button from "components/buttons/Button";
import FileDetails from "components/FileDetails";
import Checkbox from "components/forms/fields/Checkbox";
import TargetLabelSelector from "components/TargetLabelSelector";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";

import {
  CUSTOM_TARGET_OPTIONS,
  generateHelpText,
  generateSelectedLabels,
  getCustomTarget,
  getTargetType,
} from "pages/SoftwarePage/helpers";

import { generateFormValidation, getUniqueAppId } from "./helpers";

const baseClass = "software-vpp-form";

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

export interface ISoftwareVppFormData {
  selfService: boolean;
  targetType: string;
  customTarget: string;
  labelTargets: Record<string, boolean>;
  selectedApp?: IVppApp | null;
}

export interface IFormValidation {
  isValid: boolean;
  customTarget?: { isValid: boolean };
}

interface ISoftwareVppFormProps {
  labels: ILabelSummary[] | null;
  vppApps?: IVppApp[];
  softwareVppForEdit?: IAppStoreApp;
  onSubmit: (formData: ISoftwareVppFormData) => void;
  isLoading?: boolean;
  onCancel: () => void;
}

const SoftwareVppForm = ({
  labels,
  vppApps,
  softwareVppForEdit,
  onSubmit,
  isLoading = false,
  onCancel,
}: ISoftwareVppFormProps) => {
  const gitOpsModeEnabled = useContext(AppContext).config?.gitops
    .gitops_mode_enabled;

  const [formData, setFormData] = useState<ISoftwareVppFormData>(
    softwareVppForEdit
      ? {
          selfService: softwareVppForEdit.self_service || false,
          targetType: getTargetType(softwareVppForEdit),
          customTarget: getCustomTarget(softwareVppForEdit),
          labelTargets: generateSelectedLabels(softwareVppForEdit),
        }
      : {
          selectedApp: null,
          selfService: false,
          targetType: "All hosts",
          customTarget: "labelsIncludeAny",
          labelTargets: {},
        }
  );

  const [formValidation, setFormValidation] = useState<IFormValidation>({
    isValid: !!softwareVppForEdit, // Disables submit before VPP to add is selected
  });

  const onFormSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit(formData);
  };

  const onSelectApp = (app: IVppApp) => {
    if ("selectedApp" in formData) {
      const newFormData = {
        ...formData,
        selectedApp: app,
        selfService:
          app.platform === "ios" || app.platform === "ipados"
            ? false
            : formData.selfService,
      };
      setFormData(newFormData);
      setFormValidation(generateFormValidation(newFormData));
    }
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

  const isSubmitDisabled = !formValidation.isValid;

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

  const renderContent = () => {
    if (softwareVppForEdit) {
      return (
        <div className={`${baseClass}__form-fields`}>
          <FileDetails
            graphicNames="app-store"
            fileDetails={{ name: softwareVppForEdit.name, platform: "macOS" }}
            canEdit={false}
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
            dropdownHelpText={
              generateHelpText("manual", formData.customTarget) // maps to manual install help text
            }
          />
          {renderSelfServiceContent(softwareVppForEdit.platform)}
        </div>
      );
    }

    if (vppApps) {
      return (
        <div className={`${baseClass}__form-fields`}>
          <VppAppList
            apps={vppApps}
            selectedApp={formData.selectedApp || null}
            onSelect={onSelectApp}
          />
          <div className={`${baseClass}__help-text`}>
            These apps were added in Apple Business Manager (ABM). To add more
            apps, head to{" "}
            <CustomLink url="https://business.apple.com" text="ABM" newTab />
          </div>
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
            dropdownHelpText={
              generateHelpText("manual", formData.customTarget) // maps to manual install help text
            }
          />
          {renderSelfServiceContent(
            ("selectedApp" in formData &&
              formData.selectedApp &&
              formData.selectedApp.platform) ||
              ""
          )}
        </div>
      );
    }

    return null;
  };

  const contentWrapperClasses = classnames(`${baseClass}__content-wrapper`, {
    [`${baseClass}__content-disabled`]: isLoading,
  });

  const formContentClasses = classnames(`${baseClass}__form-content`, {
    [`${baseClass}__form-content--disabled`]: gitOpsModeEnabled,
  });

  return (
    <form className={baseClass} onSubmit={onFormSubmit}>
      {isLoading && <div className={`${baseClass}__overlay`} />}
      <div className={contentWrapperClasses}>
        {!softwareVppForEdit && (
          <p>Apple App Store apps purchased via Apple Business Manager:</p>
        )}
        <div className={formContentClasses}>
          <>{renderContent()}</>
        </div>
        <div className={`${baseClass}__action-buttons`}>
          <GitOpsModeTooltipWrapper
            position="bottom"
            tipOffset={8}
            renderChildren={(disableChildren) => (
              <Button
                type="submit"
                variant="brand"
                disabled={disableChildren || isSubmitDisabled}
                isLoading={isLoading}
                className={`${baseClass}__add-secret-btn`}
              >
                {softwareVppForEdit ? "Save" : "Add software"}
              </Button>
            )}
          />
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </div>
    </form>
  );
};

export default SoftwareVppForm;
