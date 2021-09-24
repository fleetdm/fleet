import React, { useState, useCallback, Component } from "react";

import Button from "components/buttons/Button";

import { IFormField } from "interfaces/form_field";
import { IQuery } from "interfaces/query";
import { IScheduledQuery } from "interfaces/scheduled_query";
import { ITarget } from "interfaces/target";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import SelectTargetsDropdown from "components/forms/fields/SelectTargetsDropdown";
import PackQueriesListWrapper from "components/queries/PackQueriesListWrapper";

const fieldNames = ["description", "name", "targets"];
const baseClass = "edit-pack-form";

interface IEditPackForm {
  className?: string;
  handleSubmit: (formData: any) => void;
  onCancelEditPack: () => void;
  onFetchTargets?: (query: IQuery, targetsResponse: any) => boolean;
  onAddPackQuery: () => void;
  onEditPackQuery: (selectedTableQueryIds: any) => void;
  onRemovePackQueries: (selectedTableQueryIds: any) => void;
  onPackQueryFormSubmit: (
    formData: IPackQueryFormData,
    editQuery: IScheduledQuery | undefined
  ) => boolean;
  packId: number;
  packTargets?: ITarget[];
  targetsCount?: number;
  isPremiumTier?: boolean;
  formData: any;
  scheduledQueries: IScheduledQuery[];
  isLoadingPackQueries: boolean;
}

interface IPackQueryFormData {
  interval: number;
  name?: string;
  shard: number;
  query?: string;
  query_id?: number;
  removed: boolean;
  snapshot: boolean;
  pack_id: number;
  platform: string;
  version: string;
}

interface IEditPackFormData {
  name: string;
  description: string;
  targets: ITarget[];
}

const EditPackForm = ({
  className,
  handleSubmit,
  onCancelEditPack,
  onFetchTargets,
  onAddPackQuery,
  onEditPackQuery,
  onRemovePackQueries,
  onPackQueryFormSubmit,
  packId,
  packTargets,
  scheduledQueries,
  isLoadingPackQueries,
  targetsCount,
  isPremiumTier,
  formData,
}: IEditPackForm): JSX.Element => {
  const [packName, setPackName] = useState<string>(formData.name);
  const [packDescription, setPackDescription] = useState<string>(
    formData.description
  );
  const [packFormTargets, setPackFormTargets] = useState<ITarget[]>(
    formData.targets
  );

  const onChangePackName = (value: string) => {
    setPackName(value);
  };

  const onChangePackDescription = (value: string) => {
    setPackDescription(value);
  };

  const onChangePackTargets = (value: ITarget[]) => {
    setPackFormTargets(value);
    console.log("value", value);
  };

  const onFormSubmit = () => {
    console.log("handle submit params", {
      name: packName,
      description: packDescription,
      targets: [...packFormTargets],
    });
    debugger;
    handleSubmit({
      name: packName,
      description: packDescription,
      targets: [...packFormTargets],
    });
  };

  return (
    <form className={`${baseClass} ${className}`} onSubmit={onFormSubmit}>
      <h1>Edit pack</h1>
      <InputField
        onChange={onChangePackName}
        value={packName}
        placeholder="Name"
        label="Name"
        inputWrapperClass={`${baseClass}__pack-title`}
      />
      <InputField
        onChange={onChangePackDescription}
        value={packDescription}
        inputWrapperClass={`${baseClass}__pack-description`}
        label="Description"
        placeholder="Add a description of your pack"
        type="textarea"
      />
      <SelectTargetsDropdown
        label="Select pack targets"
        name="selected-pack-targets"
        onFetchTargets={onFetchTargets}
        onSelect={onChangePackTargets}
        selectedTargets={packFormTargets}
        targetsCount={targetsCount}
        isPremiumTier={isPremiumTier}
      />
      <PackQueriesListWrapper
        onAddPackQuery={onAddPackQuery}
        onEditPackQuery={onEditPackQuery}
        onRemovePackQueries={onRemovePackQueries}
        onPackQueryFormSubmit={onPackQueryFormSubmit}
        scheduledQueries={scheduledQueries}
        packId={packId}
        isLoadingPackQueries={isLoadingPackQueries}
      />
      <div className={`${baseClass}__pack-buttons`}>
        <Button onClick={onCancelEditPack} type="button" variant="inverse">
          Cancel
        </Button>
        <Button type="submit" variant="brand">
          Save
        </Button>
      </div>
    </form>
  );
};

export default EditPackForm;
