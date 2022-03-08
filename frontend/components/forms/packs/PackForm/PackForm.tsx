import React, { useState } from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import { IQuery } from "interfaces/query";
import { ITarget, ITargetsAPIResponse } from "interfaces/target";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import SelectTargetsDropdown from "components/forms/fields/SelectTargetsDropdown";

const baseClass = "pack-form";

interface IPackForm {
  className?: string;
  handleSubmit: (formData: IEditPackFormData) => void;
  onFetchTargets?: (
    query: IQuery,
    targetsResponse: ITargetsAPIResponse
  ) => boolean;
  selectedTargetsCount?: number;
  isPremiumTier?: boolean;
  serverErrors: { base: string };
}

interface IEditPackFormData {
  name: string;
  description: string;
  targets: ITarget[];
}

const EditPackForm = ({
  className,
  handleSubmit,
  onFetchTargets,
  selectedTargetsCount,
  isPremiumTier,
  serverErrors,
}: IPackForm): JSX.Element => {
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
  const [packName, setPackName] = useState<string>("");
  const [packDescription, setPackDescription] = useState<string>("");
  const [packFormTargets, setPackFormTargets] = useState<ITarget[] | []>([]);

  const onChangePackName = (value: string) => {
    setPackName(value);
  };

  const onChangePackDescription = (value: string) => {
    setPackDescription(value);
  };

  const onChangePackTargets = (value: ITarget[]) => {
    setPackFormTargets(value);
  };

  const onFormSubmit = (): void => {
    if (packName === "") {
      return setErrors({
        ...errors,
        name: "Pack name must be present",
      });
    }

    return handleSubmit({
      name: packName,
      description: packDescription,
      targets: [...packFormTargets],
    });
  };

  const packFormClass = classnames(baseClass, className);

  return (
    <form className={packFormClass} onSubmit={onFormSubmit} autoComplete="off">
      <h1>New pack</h1>
      {serverErrors?.base && (
        <div className="form__base-error">{serverErrors.base}</div>
      )}
      <InputField
        onChange={onChangePackName}
        value={packName}
        placeholder="Name"
        label="Name"
        name="name"
        error={errors.name}
        inputWrapperClass={`${baseClass}__pack-title`}
      />
      <InputField
        onChange={onChangePackDescription}
        value={packDescription}
        inputWrapperClass={`${baseClass}__pack-description`}
        label="Description"
        name="description"
        placeholder="Add a description of your pack"
        type="textarea"
      />
      <div className={`${baseClass}__pack-targets`}>
        <SelectTargetsDropdown
          label="Select pack targets"
          name="selected-pack-targets"
          onFetchTargets={onFetchTargets}
          onSelect={onChangePackTargets}
          selectedTargets={packFormTargets}
          targetsCount={selectedTargetsCount}
          isPremiumTier={isPremiumTier}
        />
      </div>
      <div className={`${baseClass}__pack-buttons`}>
        <Button onClick={onFormSubmit} variant="brand">
          Save query pack
        </Button>
      </div>
    </form>
  );
};

export default EditPackForm;
