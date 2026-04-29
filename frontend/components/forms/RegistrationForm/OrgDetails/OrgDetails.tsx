import React, { useState } from "react";

import {
  ORG_LOGO_ACCEPT,
  ORG_LOGO_HELP_TEXT,
  validateOrgLogoFile,
} from "utilities/file/orgLogoFile";

import Button from "components/buttons/Button";
import FileUploader from "components/FileUploader";
import InputField from "components/forms/fields/InputField";

interface IOrgDetailsFormData {
  org_name: string;
  org_logo_file?: File | null;
}

interface IOrgDetailsErrors {
  org_name?: string;
  org_logo_file?: string;
}

interface IOrgDetailsProps {
  className?: string;
  currentPage?: boolean;
  formData?: Partial<IOrgDetailsFormData>;
  handleSubmit: (formData: IOrgDetailsFormData) => void;
}

const OrgDetails = ({
  className,
  currentPage,
  formData,
  handleSubmit,
}: IOrgDetailsProps) => {
  const [orgName, setOrgName] = useState<string>(formData?.org_name || "");
  const [orgLogoFile, setOrgLogoFile] = useState<File | null>(
    formData?.org_logo_file || null
  );
  const [errors, setErrors] = useState<IOrgDetailsErrors>({});

  const onOrgNameChange = (value: string) => {
    setOrgName(value);
    setErrors((prev) => ({ ...prev, org_name: undefined }));
  };

  const onFileSelect = async (files: FileList | null) => {
    if (!files || files.length === 0) return;
    const file = files[0];
    const result = await validateOrgLogoFile(file);
    if (!result.valid) {
      setErrors((prev) => ({ ...prev, org_logo_file: result.error }));
      return;
    }
    setErrors((prev) => ({ ...prev, org_logo_file: undefined }));
    setOrgLogoFile(file);
  };

  const onDeleteFile = () => {
    setOrgLogoFile(null);
    setErrors((prev) => ({ ...prev, org_logo_file: undefined }));
  };

  const onSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    if (!orgName) {
      setErrors((prev) => ({
        ...prev,
        org_name: "Organization name must be present",
      }));
      return;
    }
    handleSubmit({ org_name: orgName, org_logo_file: orgLogoFile });
  };

  return (
    <form onSubmit={onSubmit} className={className} autoComplete="off">
      <InputField
        label="Organization name"
        name="org_name"
        value={orgName}
        onChange={onOrgNameChange}
        error={errors.org_name}
        autofocus={!!currentPage}
      />
      <FileUploader
        label="Organization logo (optional)"
        graphicName="file-png"
        accept={ORG_LOGO_ACCEPT}
        message={ORG_LOGO_HELP_TEXT}
        buttonMessage="Choose file"
        onFileUpload={onFileSelect}
        onDeleteFile={onDeleteFile}
        canEdit
        fileDetails={
          orgLogoFile
            ? {
                name: orgLogoFile.name,
                description: "PNG",
              }
            : undefined
        }
        internalError={errors.org_logo_file}
      />
      <div className="button-wrap--center">
        <Button type="submit" disabled={!currentPage} size="wide">
          Next
        </Button>
      </div>
    </form>
  );
};

export default OrgDetails;
