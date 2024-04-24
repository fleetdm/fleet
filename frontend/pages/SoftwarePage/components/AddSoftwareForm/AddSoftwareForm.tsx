import Spinner from "components/Spinner";
import React, { useState } from "react";

const baseClass = "add-software-form";

const UploadingSoftware = () => {
  return (
    <div className={`${baseClass}__uploading-message`}>
      <Spinner centered={false} />
      <p>Uploading. It may take few minutes to finish.</p>
    </div>
  );
};

interface IAddSoftwareFormData {
  software: File;
  installScript?: string;
  preInstallQuery?: string;
  postInstallScript?: string;
}

interface IAddSoftwareFormProps {
  isUploading: boolean;
}

const AddSoftwareForm = ({ isUploading }: IAddSoftwareFormProps) => {
  const [formData, setFormData] = useState<IAddSoftwareFormData | null>(null);

  return (
    <div className={baseClass}>
      {isUploading ? <UploadingSoftware /> : <p>Form</p>}
    </div>
  );
};

export default AddSoftwareForm;
