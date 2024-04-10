import ManualLabelForm from "pages/labels/components/ManualLabelForm";
import React from "react";

const baseClass = "manual-label";

interface IManualLabelProps {}

const ManualLabel = ({}: IManualLabelProps) => {
  return (
    <div className={baseClass}>
      <ManualLabelForm />
    </div>
  );
};

export default ManualLabel;
