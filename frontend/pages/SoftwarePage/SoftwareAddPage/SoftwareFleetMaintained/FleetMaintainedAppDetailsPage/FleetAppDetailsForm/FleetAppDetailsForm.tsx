/** FleetAppDetailsForm is a separate component remnant of when we had advanced options on add <4.83 */

import React, { useState } from "react";
import { SoftwareCategory } from "interfaces/software";

import { getPathWithQueryParams } from "utilities/url";
import paths from "router/paths";

import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";
import CustomLink from "components/CustomLink";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import SoftwareOptionsSelector from "pages/SoftwarePage/components/forms/SoftwareOptionsSelector";
import {
  PatchOption,
  SoftwareDeploySelector,
} from "pages/SoftwarePage/components/forms/SoftwareDeploySelector";

const baseClass = "fleet-app-details-form";

export const softwareAlreadyAddedTipContent = (
  softwareTitleId?: number,
  teamId?: string
) => {
  const pathToSoftwareTitles = softwareTitleId
    ? getPathWithQueryParams(
        paths.SOFTWARE_TITLE_DETAILS(softwareTitleId.toString()),
        {
          fleet_id: teamId,
        }
      )
    : "";
  return (
    <>
      You already added this software.
      <br />
      <CustomLink
        url={pathToSoftwareTitles}
        text="View software"
        variant="tooltip-link"
      />
    </>
  );
};
export interface IFleetMaintainedAppFormData {
  selfService: boolean;
  forceInstall: boolean;
  patch: boolean;
  patchOption: PatchOption;
  installScript: string;
  preInstallQuery?: string;
  postInstallScript?: string;
  uninstallScript?: string;
  targetType: string;
  customTarget: string;
  labelTargets: Record<string, boolean>;
  categories: string[];
}

export interface IFormValidation {
  isValid: boolean;
  preInstallQuery?: { isValid: boolean; message?: string };
  customTarget?: { isValid: boolean };
}

interface IFleetAppDetailsFormProps {
  categories?: SoftwareCategory[] | null;
  defaultInstallScript: string;
  defaultPostInstallScript: string;
  defaultUninstallScript: string;
  teamId?: string;
  onCancel: () => void;
  onSubmit: (formData: IFleetMaintainedAppFormData) => void;
  softwareTitleId?: number;
}

const FleetAppDetailsForm = ({
  categories,
  defaultInstallScript,
  defaultPostInstallScript,
  defaultUninstallScript,
  teamId,
  onCancel,
  onSubmit,
  softwareTitleId,
}: IFleetAppDetailsFormProps) => {
  const [formData, setFormData] = useState<IFleetMaintainedAppFormData>({
    selfService: false,
    forceInstall: false,
    patch: false,
    patchOption: "closed",
    preInstallQuery: "",
    installScript: defaultInstallScript,
    postInstallScript: defaultPostInstallScript,
    uninstallScript: defaultUninstallScript,
    targetType: "All hosts",
    customTarget: "labelsIncludeAny",
    labelTargets: {},
    categories: categories || [],
  });

  const onToggleSelfService = () => {
    setFormData((prevData: IFleetMaintainedAppFormData) => ({
      ...prevData,
      selfService: !prevData.selfService,
    }));
  };

  const onSubmitForm = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit(formData);
  };

  const isSoftwareAlreadyAdded = !!softwareTitleId;
  const isSubmitDisabled = isSoftwareAlreadyAdded;

  return (
    <form className={baseClass} onSubmit={onSubmitForm}>
      <SoftwareOptionsSelector
        formData={formData}
        onToggleSelfService={onToggleSelfService}
        onClickPreviewEndUserExperience={() => undefined}
        onSelectCategory={() => undefined}
      />
      <GitOpsModeTooltipWrapper
        entityType="software"
        renderChildren={(disableChildren) => (
          <SoftwareDeploySelector
            forceInstall={formData.forceInstall}
            patch={formData.patch}
            patchOption={formData.patchOption}
            onToggleForceInstall={(forceInstall) =>
              setFormData((prevData) => ({ ...prevData, forceInstall }))
            }
            onTogglePatch={(patch) =>
              setFormData((prevData) => ({ ...prevData, patch }))
            }
            onSelectPatchOption={(patchOption) =>
              setFormData((prevData) => ({ ...prevData, patchOption }))
            }
            disabled={disableChildren}
          />
        )}
      />
      <div className={`${baseClass}__action-buttons`}>
        <GitOpsModeTooltipWrapper
          entityType="software"
          renderChildren={(disableChildren) => (
            <TooltipWrapper
              tipContent={softwareAlreadyAddedTipContent(
                softwareTitleId,
                teamId
              )}
              disableTooltip={!isSoftwareAlreadyAdded}
              position="left"
              showArrow
              underline={false}
              tipOffset={10}
            >
              <Button
                type="submit"
                disabled={disableChildren || isSubmitDisabled}
              >
                Add software
              </Button>
            </TooltipWrapper>
          )}
        />
        <Button onClick={onCancel} variant="inverse">
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default FleetAppDetailsForm;
