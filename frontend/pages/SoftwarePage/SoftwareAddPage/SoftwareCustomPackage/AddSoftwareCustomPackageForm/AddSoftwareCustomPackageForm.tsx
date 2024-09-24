import React, { useState } from "react";

import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import RevealButton from "components/buttons/RevealButton";

import AdvancedOptionsFields from "pages/SoftwarePage/components/AdvancedOptionsFields";

import { generateFormValidation } from "./helpers";

const baseClass = "add-software-custom-package-form";

export interface ICustomPackageAppFormData {
  selfService: boolean;
  installScript: string;
  preInstallQuery?: string;
  postInstallScript?: string;
  uninstallScript?: string;
}

export interface IFormValidation {
  isValid: boolean;
  preInstallQuery?: { isValid: boolean; message?: string };
}

interface IAddSoftwareCustomPackageFormProps {
  showSchemaButton: boolean;
  onClickShowSchema: () => void;
  onCancel: () => void;
  onSubmit: (formData: ICustomPackageAppFormData) => void;
}

const AddSoftwareCustomPackageForm = ({
  showSchemaButton,
  onClickShowSchema,
  onCancel,
  onSubmit,
}: IAddSoftwareCustomPackageFormProps) => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

  const [formData, setFormData] = useState<ICustomPackageAppFormData>({
    selfService: false,
    preInstallQuery: undefined,
    installScript: defaultInstallScript,
    postInstallScript: defaultPostInstallScript,
    uninstallScript: defaultUninstallScript,
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

  const onSubmitForm = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit(formData);
  };

  return (
    <form className={baseClass} onSubmit={onSubmitForm}>
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
        <AdvancedOptionsFields
          showSchemaButton={true}
          installScriptHelpText={"test"}
          postInstallScriptHelpText={"test"}
          uninstallScriptHelpText={"test"}
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
      </div>
    </form>
  );
};

export default AddSoftwareCustomPackageForm;
