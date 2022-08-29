import React, { useState, useEffect } from "react";
import { IAceEditor } from "react-ace/lib/types";
import { noop, size } from "lodash";
import { useDebouncedCallback } from "use-debounce";

import { ILabel, ILabelFormData } from "interfaces/label";
import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import FleetAce from "components/FleetAce";
// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";

interface ILabelFormProps {
  baseError: string;
  selectedLabel?: ILabel;
  isEdit?: boolean;
  onCancel: () => void;
  handleSubmit: (formData: ILabelFormData) => void;
  onOsqueryTableSelect?: (tableName: string) => void;
  backendValidators: { [key: string]: string };
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

const validateQuerySQL = (query: string) => {
  const errors: { [key: string]: any } = {};
  const { error: queryError, valid: queryValid } = validateQuery(query);

  if (!queryValid) {
    errors.query = queryError;
  }

  const valid = !size(errors);
  return { valid, errors };
};

const LabelForm = ({
  baseError,
  selectedLabel,
  isEdit,
  onCancel,
  handleSubmit,
  onOsqueryTableSelect,
  backendValidators,
}: ILabelFormProps): JSX.Element => {
  const [name, setName] = useState<string>(selectedLabel?.name || "");
  const [nameError, setNameError] = useState<string>("");
  const [description, setDescription] = useState<string>(
    selectedLabel?.description || ""
  );
  const [descriptionError, setDescriptionError] = useState<string>("");
  const [query, setQuery] = useState<string>(selectedLabel?.query || "");
  const [queryError, setQueryError] = useState<string>("");
  const [platform, setPlatform] = useState<string>(
    selectedLabel?.platform || ""
  );

  const debounceSQL = useDebouncedCallback((queryString: string) => {
    let valid = true;
    const { valid: isValidated, errors: newErrors } = validateQuerySQL(
      queryString
    );
    valid = isValidated;

    if (query === "") {
      setQueryError("");
    } else {
      setQueryError(newErrors.query);
    }
  }, 500);

  useEffect(() => {
    setNameError(backendValidators.name);
    setDescriptionError(backendValidators.description);
  }, [backendValidators]);

  useEffect(() => {
    debounceSQL(query);
  }, [query]);

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
    setNameError("");
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
        placeholder="Label name"
      />
      <InputField
        error={descriptionError}
        name="description"
        onChange={onDescriptionChange}
        value={description}
        inputClassName={`${baseClass}__label-description`}
        label="Description"
        type="textarea"
        placeholder="Label description (optional)"
      />
      {!isManual && !isEdit && (
        <div className="form-field form-field--dropdown">
          <Dropdown
            label="Platform"
            name="platform"
            onChange={onPlatformChange}
            value={platform}
            options={platformOptions}
            classname={`${baseClass}__platform-dropdown`}
            wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--platform`}
          />
        </div>
      )}
      {isEdit && platform && (
        <div className={`${baseClass}__label-platform`}>
          <p className="title">Platform</p>
          <p>{platform ? PLATFORM_STRINGS[platform] : "All platforms"}</p>
          <p className="hint">
            Label platforms are immutable. To change the platform, delete this
            label and create a new one.
          </p>
        </div>
      )}
      <div className="modal-cta-wrap">
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
