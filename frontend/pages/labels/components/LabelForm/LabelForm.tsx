import React, { ReactNode, useState } from "react";

import validate_presence from "components/forms/validators/validate_presence";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import { f } from "msw/lib/glossary-dc3fd077";

export interface ILabelFormData {
  name: string;
  description: string;
}

interface ILabelFormProps {
  defaultName?: string;
  defaultDescription?: string;
  additionalFields?: ReactNode;
  isUpdatingLabel?: boolean;
  onCancel: () => void;
  onSave: (formData: ILabelFormData, isValid: boolean) => void;
}

// const validateForm = () => {

//   return validate_presence("name");
// };

const baseClass = "label-form";

// const PLATFORM_STRINGS: { [key: string]: string } = {
//   darwin: "macOS",
//   windows: "MS Windows",
//   ubuntu: "Ubuntu Linux",
//   centos: "CentOS Linux",
// };

// const platformOptions = [
//   { label: "All platforms", value: "" },
//   { label: "macOS", value: "darwin" },
//   { label: "Windows", value: "windows" },
//   { label: "Ubuntu", value: "ubuntu" },
//   { label: "Centos", value: "centos" },
// ];

// const validateQuerySQL = (query: string) => {
//   const errors: { [key: string]: any } = {};
//   const { error: queryError, valid: queryValid } = validateQuery(query);

//   if (!queryValid) {
//     errors.query = queryError;
//   }

//   const valid = !size(errors);
//   return { valid, errors };
// };

const LabelForm = ({
  defaultName = "",
  defaultDescription = "",
  additionalFields,
  isUpdatingLabel,
  onCancel,
  onSave,
}: ILabelFormProps): JSX.Element => {
  const [name, setName] = useState(defaultName);
  const [description, setDescription] = useState(defaultDescription);
  const [nameError, setNameError] = useState<string | null>("");

  // const debounceSQL = useDebouncedCallback((queryString: string) => {
  //   let valid = true;
  //   const { valid: isValidated, errors: newErrors } = validateQuerySQL(
  //     queryString
  //   );
  //   valid = isValidated;

  //   if (query === "") {
  //     setQueryError("");
  //   } else {
  //     setQueryError(newErrors.query);
  //   }
  // }, 500);

  // useEffect(() => {
  //   debounceSQL(query);
  // }, [query]);

  // const onLoad = (editor: IAceEditor) => {
  //   editor.setOptions({
  //     enableLinking: true,
  //     enableMultiselect: false, // Disables command + click creating multiple cursors
  //   });

  //   // @ts-expect-error
  //   // the string "linkClick" is not officially in the lib but we need it
  //   editor.on("linkClick", (data) => {
  //     const { type, value } = data.token;

  //     if (type === "osquery-token" && onOsqueryTableSelect) {
  //       return onOsqueryTableSelect(value);
  //     }

  //     return false;
  //   });
  // };

  const onNameChange = (value: string) => {
    setName(value);
    setNameError(null);
  };

  const onDescriptionChange = (value: string) => {
    setDescription(value);
  };

  // const onPlatformChange = (value: string) => {
  //   setPlatform(value);
  // };

  const onSubmitForm = (evt: React.FormEvent) => {
    evt.preventDefault();

    let isFormValid = true;
    if (!validate_presence(name)) {
      setNameError("Label title must be present");
      isFormValid = false;
    }

    onSave({ name, description }, isFormValid);
  };

  // const renderLabelComponent = (): JSX.Element | null => {
  //   if (!showOpenSchemaActionText) {
  //     return null;
  //   }

  //   return (
  //     <Button variant="text-icon" onClick={onOpenSchemaSidebar}>
  //       <>
  //         <Icon name="info" size="small" />
  //         Show schema
  //       </>
  //     </Button>
  //   );
  // };

  // const isBuiltin =
  //   selectedLabel &&
  //   (selectedLabel.label_type === "builtin" || selectedLabel.type === "status");
  // const aceHelpText = isEdit
  //   ? "Label queries are immutable. To change the query, delete this label and create a new one."
  //   : "";

  // if (isBuiltin) {
  //   return (
  //     <div className={`${baseClass}__wrapper`}>
  //       <h1>Built in labels cannot be edited</h1>
  //     </div>
  //   );
  // }

  return (
    <form className={`${baseClass}__wrapper`} onSubmit={onSubmitForm}>
      {/* {!isManual && (
        <FleetAce
          error={queryError}
          name="query"
          onChange={onQueryChange}
          value={query}
          label="SQL"
          labelActionComponent={renderLabelComponent()}
          onLoad={onLoad}
          readOnly={isEdit}
          wrapperClassName={`${baseClass}__text-editor-wrapper form-field`}
          helpText={aceHelpText}
          handleSubmit={noop}
          wrapEnabled
          focus
        />
      )} */}
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
        name="description"
        onChange={onDescriptionChange}
        value={description}
        inputClassName={`${baseClass}__label-description`}
        label="Description"
        type="textarea"
        placeholder="Label description (optional)"
      />
      {additionalFields}
      {/* {!isManual && !isEdit && (
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
          <p className="help-text">
            Label platforms are immutable. To change the platform, delete this
            label and create a new one.
          </p>
        </div>
      )} */}
      <div className="button-wrap">
        <Button onClick={onCancel} variant="inverse">
          Cancel
        </Button>
        <Button type="submit" variant="brand" isLoading={isUpdatingLabel}>
          Save
        </Button>
      </div>
    </form>
  );
};

export default LabelForm;
