import React from "react";
import { noop } from "lodash";

import LabelForm from "../LabelForm";

const baseClass = "ManualLabelForm";

interface IManualLabelFormProps {}

const ManualLabelForm = ({}: IManualLabelFormProps) => {
  return (
    <div className={baseClass}>
      <LabelForm onCancel={noop} onSave={noop} />
    </div>
  );
};

export default ManualLabelForm;
