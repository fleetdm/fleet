import React from "react";

import LabelForm from "../LabelForm";
import { ILabelFormData } from "../LabelForm/LabelForm";

const baseClass = "ManualLabelForm";

export interface IManualLabelFormData {
  name: string;
  description: string;
  hosts: string[];
}

interface IManualLabelFormProps {
  defaultQuery?: string;
  defaultPlatform?: string;
  isEditing?: boolean;
  onSave: (formData: IManualLabelFormData) => void;
  onCancel: () => void;
}

const ManualLabelForm = ({ onSave, onCancel }: IManualLabelFormProps) => {
  const onSaveNewLabel = (
    formData: ILabelFormData,
    labelFormDataValid: boolean
  ) => {
    console.log("data", formData);
  };

  return (
    <div className={baseClass}>
      <LabelForm
        onCancel={onCancel}
        onSave={onSaveNewLabel}
        additionalFields={<p>test</p>}
      />
    </div>
  );
};

export default ManualLabelForm;
