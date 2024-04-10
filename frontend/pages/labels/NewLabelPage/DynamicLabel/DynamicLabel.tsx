import React from "react";
import { RouteComponentProps } from "react-router";
import { noop } from "lodash";

import DynamicLabelForm from "pages/labels/components/DynamicLabelForm";
import { IDynamicLabelFormData } from "pages/labels/components/DynamicLabelForm/DynamicLabelForm";

const baseClass = "dynamic-label";

type IDynamicLabelProps = RouteComponentProps<never, never>;

const DynamicLabel = ({}: IDynamicLabelProps) => {
  const onSaveNewLabel = (formData: IDynamicLabelFormData) => {
    console.log("data", formData);
  };

  return (
    <div className={baseClass}>
      <DynamicLabelForm onSave={onSaveNewLabel} onCancel={noop} />
    </div>
  );
};

export default DynamicLabel;
