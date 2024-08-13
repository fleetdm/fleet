import React, { useCallback, useContext, useState } from "react";

import PATHS from "router/paths";

import { NotificationContext } from "context/notification";
import { getFileDetails } from "utilities/file/fileUtils";
import getInstallScript from "utilities/software_install_scripts";

import { ILabel } from "interfaces/label";
import { InstallType } from "interfaces/software";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import {
  FileUploader,
  FileDetails,
} from "components/FileUploader/FileUploader";
import Spinner from "components/Spinner";
import TooltipWrapper from "components/TooltipWrapper";
import Radio from "components/forms/fields/Radio";
import Card from "components/Card";
import CustomLink from "components/CustomLink";

import AddPackageAdvancedOptions from "../AddPackageAdvancedOptions";

import {
  generateFormValidation,
  INSTALL_TYPE_OPTIONS,
  LABEL_HELP_TEXT_CONFIG,
  LABEL_TARGET_MODES,
} from "./helpers";

export const baseClass = "add-package-form";

const UploadingSoftware = () => {
  return (
    <div className={`${baseClass}__uploading-message`}>
      <Spinner centered={false} />
      <p>Uploading. It may take a few minutes to finish.</p>
    </div>
  );
};

const NoLabelsCard = () => {
  return (
    <Card borderRadiusSize="medium">
      <div className={`${baseClass}__no-fleet-labels`}>
        <p className={`${baseClass}__no-fleet-labels-title`}>
          <b>No labels exist in Fleet</b>
        </p>
        <p className={`${baseClass}__no-fleet-labels-description`}>
          Add label to target specific hosts.
        </p>
        <CustomLink
          url={PATHS.LABEL_NEW}
          text="Add label"
          className={`${baseClass}__add-fleet-label-link`}
        />
      </div>
    </Card>
  );
};
export interface IAddPackageFormData {
  software: File | null;
  installScript: string;
  preInstallCondition?: string;
  postInstallScript?: string;
  selfService: boolean;
  installType: InstallType;
  includeAnyLabels: boolean;
  selectedLabels: Record<string, boolean>; // label_name: isSelected
}

export interface IFormValidation {
  isValid: boolean;
  software: { isValid: boolean };
  preInstallCondition?: { isValid: boolean; message?: string };
  postInstallScript?: { isValid: boolean; message?: string };
  selfService?: { isValid: boolean };
}

interface IAddPackageFormProps {
  isUploading: boolean;
  onCancel: () => void;
  onSubmit: (formData: IAddPackageFormData) => void;
  customLabels?: ILabel[];
}

const AddPackageForm = ({
  isUploading,
  onCancel,
  onSubmit,
  customLabels,
}: IAddPackageFormProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [showPreInstallCondition, setShowPreInstallCondition] = useState(false);
  const [showPostInstallScript, setShowPostInstallScript] = useState(false);
  const [useCustomTargets, setUseCustomTargets] = useState(false);
  const [formData, setFormData] = useState<IAddPackageFormData>({
    software: null,
    installScript: "",
    preInstallCondition: undefined,
    postInstallScript: undefined,
    selfService: false,
    installType: "manual",
    selectedLabels: {},
    includeAnyLabels: true,
  });
  const [formValidation, setFormValidation] = useState<IFormValidation>({
    isValid: false,
    software: { isValid: false },
  });

  const onFileUpload = (files: FileList | null) => {
    if (files && files.length > 0) {
      const file = files[0];

      let installScript: string;
      try {
        installScript = getInstallScript(file.name);
      } catch (e) {
        renderFlash("error", `${e}`);
        return;
      }

      const newData = {
        ...formData,
        software: file,
        installScript,
      };
      setFormData(newData);
      setFormValidation(
        generateFormValidation(
          newData,
          showPreInstallCondition,
          showPostInstallScript
        )
      );
    }
  };

  const onFormSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit(formData);
  };

  const onUpdateSelectedLabels = useCallback(
    ({ name, value }: { name: string; value: boolean }) => {
      const newSelectedLabels = { ...formData.selectedLabels, [name]: value };
      setFormData({ ...formData, selectedLabels: newSelectedLabels });
    },
    [formData]
  );

  const onTogglePreInstallConditionCheckbox = useCallback(
    (value: boolean) => {
      const newData = { ...formData, preInstallCondition: undefined };
      setShowPreInstallCondition(value);
      setFormData(newData);
      setFormValidation(
        generateFormValidation(newData, value, showPostInstallScript)
      );
    },
    [formData, showPostInstallScript]
  );

  const onTogglePostInstallScriptCheckbox = useCallback(
    (value: boolean) => {
      const newData = { ...formData, postInstallScript: undefined };
      setShowPostInstallScript(value);
      setFormData(newData);
      setFormValidation(
        generateFormValidation(newData, showPreInstallCondition, value)
      );
    },
    [formData, showPreInstallCondition]
  );

  const onChangeInstallScript = useCallback(
    (value: string) => {
      setFormData({ ...formData, installScript: value });
    },
    [formData]
  );

  const onChangePreInstallCondition = useCallback(
    (value?: string) => {
      const newData = { ...formData, preInstallCondition: value };
      setFormData(newData);
      setFormValidation(
        generateFormValidation(
          newData,
          showPreInstallCondition,
          showPostInstallScript
        )
      );
    },
    [formData, showPostInstallScript, showPreInstallCondition]
  );

  const onChangePostInstallScript = useCallback(
    (value?: string) => {
      const newData = { ...formData, postInstallScript: value };
      setFormData(newData);
      setFormValidation(
        generateFormValidation(
          newData,
          showPreInstallCondition,
          showPostInstallScript
        )
      );
    },
    [formData, showPostInstallScript, showPreInstallCondition]
  );

  const onChangeInstallType = useCallback(
    (value: InstallType) => {
      const newData = { ...formData, installType: value };
      setFormData(newData);
    },
    [formData]
  );

  const onToggleSelfServiceCheckbox = useCallback(
    (value: boolean) => {
      const newData = { ...formData, selfService: value };
      setFormData(newData);
      setFormValidation(
        generateFormValidation(
          newData,
          showPreInstallCondition,
          showPostInstallScript
        )
      );
    },
    [formData, showPostInstallScript, showPreInstallCondition]
  );

  const onChangeTargets = useCallback(
    (val: string) => {
      setUseCustomTargets(val === "custom");
    },
    [setUseCustomTargets]
  );

  const onChangeLabelTargetMode = useCallback(
    (val: string) => {
      setFormData({ ...formData, includeAnyLabels: val === "include" });
    },
    [formData]
  );

  const noRequiredLabelTargets =
    useCustomTargets && !Object.values(formData.selectedLabels).some(Boolean);

  const isSubmitDisabled = !formValidation.isValid || noRequiredLabelTargets;

  const renderLabels = () => (
    <div className={`${baseClass}__fleet-label-selection`}>
      {customLabels?.map((label) => (
        <div className={`${baseClass}__fleet-label`} key={label.name}>
          <Checkbox
            className={`${baseClass}__checkbox`}
            name={label.name}
            value={!!formData.selectedLabels[label.name]}
            onChange={onUpdateSelectedLabels}
            parseTarget
          />
          <div className={`${baseClass}__fleet-label-name`}>{label.name}</div>
        </div>
      ))}
    </div>
  );

  const renderLabelsSection = () => {
    if (!customLabels?.length) {
      return <NoLabelsCard />;
    }
    return (
      <>
        <Dropdown
          value={formData.includeAnyLabels ? "include" : "exclude"}
          options={LABEL_TARGET_MODES}
          searchable={false}
          onChange={onChangeLabelTargetMode}
        />
        <div className={`${baseClass}__include-any-toggle-text`}>
          {
            LABEL_HELP_TEXT_CONFIG[
              formData.includeAnyLabels ? "include" : "exclude"
            ][formData.installType]
          }
        </div>
        {renderLabels()}
      </>
    );
  };

  return (
    <div className={baseClass}>
      {isUploading ? (
        <UploadingSoftware />
      ) : (
        <form className={`${baseClass}__form`} onSubmit={onFormSubmit}>
          <FileUploader
            graphicName={"file-pkg"}
            accept=".pkg,.msi,.exe,.deb"
            message=".pkg, .msi, .exe, or .deb"
            onFileUpload={onFileUpload}
            buttonMessage="Choose file"
            buttonType="link"
            className={`${baseClass}__file-uploader`}
            filePreview={
              formData.software && (
                <FileDetails details={getFileDetails(formData.software)} />
              )
            }
          />
          <Dropdown
            label="Install"
            value={formData.installType}
            options={INSTALL_TYPE_OPTIONS}
            searchable={false}
            onChange={onChangeInstallType}
          />
          <Checkbox
            value={formData.selfService}
            onChange={onToggleSelfServiceCheckbox}
          >
            <TooltipWrapper
              tipContent={
                <>
                  End users can install from{" "}
                  <b>Fleet Desktop {">"} Self-service</b>.
                </>
              }
            >
              Self-service
            </TooltipWrapper>
          </Checkbox>
          <div className={`form-field ${baseClass}__target`}>
            <div className="form-field__label">Target</div>
            <Radio
              className={`${baseClass}__radio-input`}
              label="All hosts"
              id="all-hosts-target"
              checked={!useCustomTargets}
              value="all"
              onChange={onChangeTargets}
            />
            <Radio
              className={`${baseClass}__radio-input`}
              label="Custom"
              id="custom-target"
              checked={useCustomTargets}
              value="custom"
              onChange={onChangeTargets}
            />
          </div>
          {useCustomTargets && renderLabelsSection()}
          <AddPackageAdvancedOptions
            errors={{
              preInstallCondition: formValidation.preInstallCondition?.message,
              postInstallScript: formValidation.postInstallScript?.message,
            }}
            showPreInstallCondition={showPreInstallCondition}
            showInstallScript={!!formData.software}
            showPostInstallScript={showPostInstallScript}
            preInstallCondition={formData.preInstallCondition}
            postInstallScript={formData.postInstallScript}
            onTogglePreInstallCondition={onTogglePreInstallConditionCheckbox}
            onTogglePostInstallScript={onTogglePostInstallScriptCheckbox}
            onChangePreInstallCondition={onChangePreInstallCondition}
            onChangeInstallScript={onChangeInstallScript}
            onChangePostInstallScript={onChangePostInstallScript}
            installScript={formData.installScript}
          />
          <div className="modal-cta-wrap">
            <Button type="submit" variant="brand" disabled={isSubmitDisabled}>
              Add software
            </Button>
            <Button onClick={onCancel} variant="inverse">
              Cancel
            </Button>
          </div>
        </form>
      )}
    </div>
  );
};

export default AddPackageForm;
