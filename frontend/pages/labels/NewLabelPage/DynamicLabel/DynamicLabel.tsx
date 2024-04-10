import React from "react";
import { RouteComponentProps } from "react-router";

import DynamicLabelForm from "pages/labels/components/DynamicLabelForm";
import { IDynamicLabelFormData } from "pages/labels/components/DynamicLabelForm/DynamicLabelForm";

const baseClass = "dynamic-label";

type IDynamicLabelProps = RouteComponentProps<never, never>;

const DynamicLabel = ({ router }: IDynamicLabelProps) => {
  const onSaveNewLabel = (formData: IDynamicLabelFormData) => {
    console.log("data", formData);
  };

  const onCancelLabel = () => {
    router.goBack();
  };

  return (
    <div className={baseClass}>
      <DynamicLabelForm onSave={onSaveNewLabel} onCancel={onCancelLabel} />
    </div>
  );
};

export default DynamicLabel;
