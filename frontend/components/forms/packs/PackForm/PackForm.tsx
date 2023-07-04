import React, { useState } from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import { IQuery } from "interfaces/query";
import { ITarget, ITargetsAPIResponse } from "interfaces/target";
import { IEditPackFormData } from "interfaces/pack";
import PATHS from "router/paths";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import BackLink from "components/BackLink";
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
  isUpdatingPack: boolean;
}

const EditPackForm = ({
  className,
  handleSubmit,
  onFetchTargets,
  selectedTargetsCount,
  isPremiumTier,
  isUpdatingPack,
}: IPackForm): JSX.Element => {
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
  const [packName, setPackName] = useState("");
  const [packDescription, setPackDescription] = useState("");
  const [packFormTargets, setPackFormTargets] = useState<ITarget[] | []>([]);

  const onChangePackName = (value: string) => {
    setPackName(value);
    setErrors({});
  };

  const onChangePackDescription = (value: string) => {
    setPackDescription(value);
  };

  const onChangePackTargets = (value: ITarget[]) => {
    setPackFormTargets(value);
  };

  const onFormSubmit = (evt: React.FormEvent<HTMLFormElement>): void => {
    evt.preventDefault();

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
    <div className={`${baseClass}__form`}>
      <div className={`${baseClass}__header-links`}>
        <BackLink text="Back to packs" path={PATHS.MANAGE_PACKS} />
      </div>
      <form
        className={packFormClass}
        onSubmit={onFormSubmit}
        autoComplete="off"
      >
        <h1>New pack</h1>
        <InputField
          onChange={onChangePackName}
          value={packName}
          placeholder="Name"
          label="Name"
          name="name"
          error={errors.name}
          inputWrapperClass={`${baseClass}__pack-title`}
          autofocus
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
          <Button type="submit" variant="brand" isLoading={isUpdatingPack}>
            Save query pack
          </Button>
        </div>
      </form>
    </div>
  );
};

export default EditPackForm;
