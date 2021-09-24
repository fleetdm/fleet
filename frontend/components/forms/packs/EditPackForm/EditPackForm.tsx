import React, { Component } from "react";
import PropTypes from "prop-types";

import Button from "components/buttons/Button";
// @ts-ignore
import Form from "components/forms/Form";
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
  handleSubmit?: (formData: any) => void;
  onCancelEditPack: () => void;
  onFetchTargets?: (query: IQuery, targetsResponse: any) => boolean;
  onAddPackQuery: () => void;
  onEditPackQuery: () => void;
  onRemovePackQueries: () => void;
  onPackQueryFormSubmit: (formData: any) => void;
  packId: number;
  packTargets?: ITarget[];
  targetsCount?: number;
  isPremiumTier?: boolean;
  fields: { description: IFormField; name: IFormField; targets: IFormField };
  scheduledQueries: IScheduledQuery[];
  isLoadingPackQueries: boolean;
}
const EditPackForm = (props: IEditPackForm): JSX.Element => {
  const {
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
    fields,
  } = props;

  return (
    <form className={`${baseClass} ${className}`} onSubmit={handleSubmit}>
      <h1>Edit pack</h1>
      <InputField
        {...fields.name}
        placeholder="Name"
        label="Name"
        inputWrapperClass={`${baseClass}__pack-title`}
      />
      <InputField
        {...fields.description}
        inputWrapperClass={`${baseClass}__pack-description`}
        label="Description"
        placeholder="Add a description of your pack"
        type="textarea"
      />
      <SelectTargetsDropdown
        {...fields.targets}
        label="Select pack targets"
        name="selected-pack-targets"
        onFetchTargets={onFetchTargets}
        onSelect={fields.targets.onChange}
        selectedTargets={fields.targets.value}
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

export default Form(EditPackForm, {
  fields: fieldNames,
});
