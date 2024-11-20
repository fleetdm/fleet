import React, { useState } from "react";

import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import RevealButton from "components/buttons/RevealButton";
import Button from "components/buttons/Button";
import Radio from "components/forms/fields/Radio";

import AdvancedOptionsFields from "pages/SoftwarePage/components/AdvancedOptionsFields";

import { generateFormValidation } from "./helpers";

const baseClass = "fleet-app-details-form";

export interface IFleetMaintainedAppFormData {
  selfService: boolean;
  installScript: string;
  preInstallQuery?: string;
  postInstallScript?: string;
  uninstallScript?: string;
  installType: string;
}

export interface IFormValidation {
  isValid: boolean;
  preInstallQuery?: { isValid: boolean; message?: string };
}

interface IFleetAppDetailsFormProps {
  defaultInstallScript: string;
  defaultPostInstallScript: string;
  defaultUninstallScript: string;
  showSchemaButton: boolean;
  onClickShowSchema: () => void;
  onCancel: () => void;
  onSubmit: (formData: IFleetMaintainedAppFormData) => void;
}

const FleetAppDetailsForm = ({
  defaultInstallScript,
  defaultPostInstallScript,
  defaultUninstallScript,
  showSchemaButton,
  onClickShowSchema,
  onCancel,
  onSubmit,
}: IFleetAppDetailsFormProps) => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

  const [formData, setFormData] = useState<IFleetMaintainedAppFormData>({
    selfService: false,
    preInstallQuery: undefined,
    installScript: defaultInstallScript,
    postInstallScript: defaultPostInstallScript,
    uninstallScript: defaultUninstallScript,
    installType: "manual",
  });
  const [formValidation, setFormValidation] = useState<IFormValidation>({
    isValid: true,
    preInstallQuery: { isValid: false },
  });

  const onChangePreInstallQuery = (value?: string) => {
    const newData = { ...formData, preInstallQuery: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onChangeInstallScript = (value: string) => {
    const newData = { ...formData, installScript: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onChangePostInstallScript = (value?: string) => {
    const newData = { ...formData, postInstallScript: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onChangeUninstallScript = (value?: string) => {
    const newData = { ...formData, uninstallScript: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onToggleSelfServiceCheckbox = (value: boolean) => {
    const newData = { ...formData, selfService: value };
    setFormData(newData);
    setFormValidation(generateFormValidation(newData));
  };

  const onChangeInstallType = (value: string) => {
    const newData = { ...formData, installType: value };
    setFormData(newData);
  };

  const onSubmitForm = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit(formData);
  };

  const isSubmitDisabled = !formValidation.isValid;

  return (
    <form className={baseClass} onSubmit={onSubmitForm}>
      <fieldset>
        <legend>Install</legend>
        <div className={`${baseClass}__radio-inputs`}>
          <Radio
            checked={formData.installType === "manual"}
            id="manual"
            value="manual"
            name="install-type"
            label="Manual"
            onChange={onChangeInstallType}
            helpText="Manually install on Host details page for each host."
          />
          <Radio
            checked={formData.installType === "automatic"}
            id="automatic"
            value="automatic"
            name="install-type"
            label="Automatic"
            onChange={onChangeInstallType}
            helpText={
              <>
                Automatically install on each host that&apos;s{" "}
                <TooltipWrapper tipContent="If the host already has any version of this software, it won't be installed.">
                  missing this software.
                </TooltipWrapper>{" "}
                Policy that triggers install can be customized after software is
                added.
              </>
            }
          />
        </div>
      </fieldset>
      <Checkbox
        value={formData.selfService}
        onChange={onToggleSelfServiceCheckbox}
      >
        <TooltipWrapper
          tipContent={
            <>
              End users can install from <b>Fleet Desktop {">"} Self-service</b>
              .
            </>
          }
        >
          Self-service
        </TooltipWrapper>
      </Checkbox>
      <div className={`${baseClass}__advanced-options-section`}>
        <RevealButton
          className={`${baseClass}__accordion-title`}
          isShowing={showAdvancedOptions}
          showText="Advanced options"
          hideText="Advanced options"
          caretPosition="after"
          onClick={() => setShowAdvancedOptions(!showAdvancedOptions)}
        />
        {showAdvancedOptions && (
          <AdvancedOptionsFields
            className={`${baseClass}__advanced-options-fields`}
            showSchemaButton={showSchemaButton}
            installScriptHelpText="Use the $INSTALLER_PATH variable to point to the installer. Currently, shell scripts are supported."
            postInstallScriptHelpText="Currently, shell scripts are supported."
            uninstallScriptHelpText="Currently, shell scripts are supported."
            errors={{
              preInstallQuery: formValidation.preInstallQuery?.message,
            }}
            preInstallQuery={formData.preInstallQuery}
            installScript={formData.installScript}
            postInstallScript={formData.postInstallScript}
            uninstallScript={formData.uninstallScript}
            onClickShowSchema={onClickShowSchema}
            onChangePreInstallQuery={onChangePreInstallQuery}
            onChangeInstallScript={onChangeInstallScript}
            onChangePostInstallScript={onChangePostInstallScript}
            onChangeUninstallScript={onChangeUninstallScript}
          />
        )}
      </div>
      <div className={`${baseClass}__form-buttons`}>
        <Button type="submit" variant="brand" disabled={isSubmitDisabled}>
          Add software
        </Button>
        <Button onClick={onCancel} variant="inverse">
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default FleetAppDetailsForm;
