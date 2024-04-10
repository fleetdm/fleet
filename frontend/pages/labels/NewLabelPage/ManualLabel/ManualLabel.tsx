import React from "react";
import { RouteComponentProps } from "react-router";

import ManualLabelForm from "pages/labels/components/ManualLabelForm";
import { IManualLabelFormData } from "pages/labels/components/ManualLabelForm/ManualLabelForm";

const baseClass = "manual-label";

type IManualLabelProps = RouteComponentProps<never, never>;

const ManualLabel = ({ router }: IManualLabelProps) => {
  const onSaveNewLabel = (formData: IManualLabelFormData) => {
    console.log("data", formData);
  };

  const onCancelLabel = () => {
    router.goBack();
  };

  return (
    <div className={baseClass}>
      <ManualLabelForm onSave={onSaveNewLabel} onCancel={onCancelLabel} />
    </div>
  );
};

export default ManualLabel;
