import React, { useEffect, useRef, useState } from "react";

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
  currentPage: boolean;
  formData?: Partial<IOrgDetailsFormData>;
  handleSubmit: (formData: IOrgDetailsFormData) => void;
}

interface ICustomLogo {
  file: File;
  url: string;
}

const OrgDetails = ({
  className,
  currentPage,
  formData,
  handleSubmit,
}: IOrgDetailsProps) => {
  const [orgName, setOrgName] = useState<string>(formData?.org_name || "");
  const [customLogo, setCustomLogo] = useState<ICustomLogo | null>(() => {
    if (!formData?.org_logo_file) return null;
    return {
      file: formData.org_logo_file,
      url: URL.createObjectURL(formData.org_logo_file),
    };
  });
  const [errors, setErrors] = useState<IOrgDetailsErrors>({});

  // Revoke every live blob URL on unmount
  const activePreviewUrlRef = useRef<string | null>(customLogo?.url ?? null);
  useEffect(() => {
    activePreviewUrlRef.current = customLogo?.url ?? null;
  }, [customLogo]);
  useEffect(
    () => () => {
      if (activePreviewUrlRef.current) {
        URL.revokeObjectURL(activePreviewUrlRef.current);
      }
    },
    []
  );

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
    const url = URL.createObjectURL(file);
    setCustomLogo((prev) => {
      if (prev) URL.revokeObjectURL(prev.url);
      return { file, url };
    });
  };

  const onDeleteFile = () => {
    setCustomLogo((prev) => {
      if (prev) URL.revokeObjectURL(prev.url);
      return null;
    });
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
    handleSubmit({
      org_name: orgName,
      org_logo_file: customLogo?.file ?? null,
    });
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
        graphicName="fleet-logo"
        accept={ORG_LOGO_ACCEPT}
        message={ORG_LOGO_HELP_TEXT}
        buttonMessage="Choose file"
        onFileUpload={onFileSelect}
        onDeleteFile={onDeleteFile}
        canEdit
        fileDetails={customLogo ? { name: customLogo.file.name } : undefined}
        customPreview={
          customLogo ? (
            <img
              src={customLogo.url}
              alt="Organization logo preview"
              width={40}
              height={40}
              style={{ objectFit: "contain" }}
            />
          ) : undefined
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
