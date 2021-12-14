import React, { useState } from "react";
import { IAceEditor } from "react-ace/lib/types";
import { noop } from "lodash";

import { ILabel, ILabelFormData } from "interfaces/label";
import Button from "components/buttons/Button"; // @ts-ignore
import Dropdown from "components/forms/fields/Dropdown"; // @ts-ignore
import InputField from "components/forms/fields/InputField";
import FleetAce from "components/FleetAce"; // @ts-ignore
import validateQuery from "components/forms/validators/validate_query";

interface ILabelFormProps {
  baseError: string;
  selectedLabel?: ILabel;
  isEdit?: boolean;
  onCancel: () => void;
  handleSubmit: (formData: ILabelFormData) => Promise<void>;
  onOsqueryTableSelect?: (tableName: string) => void;
}

const baseClass = "label-form";

const PLATFORM_STRINGS: { [key: string]: string } = {
  darwin: "macOS",
  windows: "MS Windows",
  ubuntu: "Ubuntu Linux",
  centos: "CentOS Linux",
};

const platformOptions = [
  { label: "All platforms", value: "" },
  { label: "macOS", value: "darwin" },
  { label: "Windows", value: "windows" },
  { label: "Ubuntu", value: "ubuntu" },
  { label: "Centos", value: "centos" },
];

const LabelForm = ({
  baseError,
  selectedLabel,
  isEdit,
  onCancel,
  handleSubmit,
  onOsqueryTableSelect,
}: ILabelFormProps) => {
  const [name, setName] = useState<string>(selectedLabel?.name || "");
  const [nameError, setNameError] = useState<string>("");
  const [description, setDescription] = useState<string>(
    selectedLabel?.description || ""
  );
  const [query, setQuery] = useState<string>(selectedLabel?.query || "");
  const [queryError, setQueryError] = useState<string>("");
  const [platform, setPlatform] = useState<string>(
    selectedLabel?.platform || ""
  );

  const onLoad = (editor: IAceEditor) => {
    editor.setOptions({
      enableLinking: true,
    });

    // @ts-expect-error
    // the string "linkClick" is not officially in the lib but we need it
    editor.on("linkClick", (data) => {
      const { type, value } = data.token;

      if (type === "osquery-token" && onOsqueryTableSelect) {
        return onOsqueryTableSelect(value);
      }

      return false;
    });
  };

  const onQueryChange = (value: string) => {
    setQuery(value);
  };

  const onNameChange = (value: string) => {
    setName(value);
  };

  const onDescriptionChange = (value: string) => {
    setDescription(value);
  };

  const onPlatformChange = (value: string) => {
    setPlatform(value);
  };

  const submitForm = (evt: React.FormEvent) => {
    evt.preventDefault();

    const { error, valid } = validateQuery(query);
    if (!valid) {
      setQueryError(error);
      return false;
    }

    setQueryError("");

    if (!name) {
      setNameError("Label title must be present");
      return false;
    }

    setNameError("");
    handleSubmit({
      name,
      query,
      description,
      platform,
    });
  };

  const isBuiltin =
    selectedLabel &&
    (selectedLabel.label_type === "builtin" || selectedLabel.type === "status");
  const isManual =
    selectedLabel && selectedLabel.label_membership_type === "manual";
  const headerText = isEdit ? "Edit label" : "New label";
  const saveBtnText = isEdit ? "Update label" : "Save label";
  const aceHintText = isEdit
    ? "Label queries are immutable. To change the query, delete this label and create a new one."
    : "";

  if (isBuiltin) {
    return (
      <div className={`${baseClass}__wrapper`}>
        <h1>Built in labels cannot be edited</h1>
      </div>
    );
  }

  return (
    <form
      className={`${baseClass}__wrapper`}
      onSubmit={submitForm}
      autoComplete="off"
    >
      <h1>{headerText}</h1>
      {!isManual && (
        <FleetAce
          error={queryError}
          name="query"
          onChange={onQueryChange}
          value={query}
          label="SQL"
          onLoad={onLoad}
          readOnly={isEdit}
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          hint={aceHintText}
          handleSubmit={noop}
        />
      )}

      {baseError && <div className="form__base-error">{baseError}</div>}
      <InputField
        error={nameError}
        name="name"
        onChange={onNameChange}
        value={name}
        inputClassName={`${baseClass}__label-title`}
        label="Name"
      />
      <InputField
        name="description"
        onChange={onDescriptionChange}
        value={description}
        inputClassName={`${baseClass}__label-description`}
        label="Description"
        type="textarea"
      />
      {!isManual && !isEdit && (
        <div className="form-field form-field--dropdown">
          <label className="form-field__label" htmlFor="platform">
            Platform
          </label>
          <Dropdown
            name="platform"
            onChange={onPlatformChange}
            value={platform}
            options={platformOptions}
          />
        </div>
      )}
      {isEdit && platform && (
        <div className={`${baseClass}__label-platform`}>
          <p className="title">Platform</p>
          <p>{!platform ? "All platforms" : PLATFORM_STRINGS[platform]}</p>
          <p className="hint">
            Label platforms are immutable. To change the platform, delete this
            label and create a new one.
          </p>
        </div>
      )}
      <div className={`${baseClass}__button-wrap`}>
        <Button
          className={`${baseClass}__cancel-btn`}
          onClick={onCancel}
          variant="inverse"
        >
          Cancel
        </Button>
        <Button
          className={`${baseClass}__save-btn`}
          type="submit"
          variant="brand"
        >
          {saveBtnText}
        </Button>
      </div>
    </form>
  );
};

export default LabelForm;
