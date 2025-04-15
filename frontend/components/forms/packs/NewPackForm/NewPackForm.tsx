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

const baseClass = "new-pack-form";

interface INewPackForm {
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

const NewPackForm = ({
  className,
  handleSubmit,
  onFetchTargets,
  selectedTargetsCount,
  isPremiumTier,
  isUpdatingPack,
}: INewPackForm): JSX.Element => {
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
  const [packName, setPackName] = useState("");
  const [packDescription, setPackDescription] = useState("");
  const [newPackFormTargets, setNewPackFormTargets] = useState<ITarget[] | []>(
    []
  );

  const onChangePackName = (value: string) => {
    setPackName(value);
    setErrors({});
  };

  const onChangePackDescription = (value: string) => {
    setPackDescription(value);
  };

  const onChangePackTargets = (value: ITarget[]) => {
    setNewPackFormTargets(value);
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
      targets: [...newPackFormTargets],
    });
  };

  const newPackFormClass = classnames(baseClass, className);

  return (
    <>
      <div className={`${baseClass}__header-links`}>
        <BackLink text="Back to packs" path={PATHS.MANAGE_PACKS} />
      </div>
      <form
        className={newPackFormClass}
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
        <SelectTargetsDropdown
          label="Select pack targets"
          name="selected-pack-targets"
          onFetchTargets={onFetchTargets}
          onSelect={onChangePackTargets}
          selectedTargets={newPackFormTargets}
          targetsCount={selectedTargetsCount}
          isPremiumTier={isPremiumTier}
        />
        <div className={`${baseClass}__pack-buttons`}>
          <Button type="submit" isLoading={isUpdatingPack}>
            Save query pack
          </Button>
        </div>
      </form>
    </>
  );
};

export default NewPackForm;
