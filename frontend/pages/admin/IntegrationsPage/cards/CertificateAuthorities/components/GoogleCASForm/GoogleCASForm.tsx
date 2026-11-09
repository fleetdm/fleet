import React from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import InputField from "components/forms/fields/InputField";
import useFormValidation from "hooks/useFormValidation";
import { ICertificateAuthorityPartial } from "interfaces/certificates";
import { MAX_ENTITY_CHAR_LENGTH } from "utilities/constants";

import validateGoogleCASForm from "./helpers";

const baseClass = "google-cas-form";

export interface IGoogleCASFormData {
  name: string;
  projectId: string;
  caPool: string;
  location: string;
  apiKeyJson: string;
}

const EMPTY_FORM_DATA: IGoogleCASFormData = {
  name: "",
  projectId: "",
  caPool: "",
  location: "",
  apiKeyJson: "",
};

interface IGoogleCASFormProps {
  certAuthorities?: ICertificateAuthorityPartial[];
  /** The saved CA's values. Passing them makes this the edit form. */
  defaultData?: IGoogleCASFormData;
  submitBtnText: string;
  isSubmitting: boolean;
  onSubmit: (formData: IGoogleCASFormData) => void | Promise<unknown>;
  onCancel: () => void;
}

const GoogleCASForm = ({
  certAuthorities = [],
  defaultData,
  submitBtnText,
  isSubmitting: isSubmittingProp,
  onSubmit,
  onCancel,
}: IGoogleCASFormProps) => {
  const isEditing = defaultData !== undefined;

  const {
    getTextFieldProps,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IGoogleCASFormData>({
    initialFormData: defaultData ?? EMPTY_FORM_DATA,
    validate: (formData) =>
      validateGoogleCASForm(formData, { certAuthorities, isEditing }),
    isSubmitting: isSubmittingProp,
  });

  return (
    <form className={baseClass} onSubmit={handleSubmit(onSubmit)}>
      <InputField
        {...getTextFieldProps("name")}
        label="Name"
        helpText="Letters, numbers, and underscores only."
        disabled={isSubmitting}
        placeholder="WIFI_CERTIFICATE"
        inputOptions={{ maxLength: MAX_ENTITY_CHAR_LENGTH }}
      />
      <InputField
        {...getTextFieldProps("projectId")}
        label="Project ID"
        helpText="The Google Cloud project ID."
        disabled={isSubmitting}
        placeholder="my-project"
      />
      <InputField
        {...getTextFieldProps("caPool")}
        label="CA pool"
        helpText={
          <>
            You can find your <b>CA pool</b> in the{" "}
            <CustomLink
              text="CA pool manager"
              url="https://console.cloud.google.com/security/cas/caPools"
              newTab
            />
          </>
        }
        disabled={isSubmitting}
        placeholder="my-ca-pool"
      />
      <InputField
        {...getTextFieldProps("location")}
        label="Location"
        disabled={isSubmitting}
        placeholder="us-central1"
      />
      <InputField
        {...getTextFieldProps("apiKeyJson")}
        type="password"
        label="Service account key"
        blockAutoComplete
        helpText={
          <>
            The service account key with the{" "}
            <b>CA Service Certificate Manager</b> role.{" "}
            <CustomLink
              text="Google Cloud's Service accounts"
              url="https://console.cloud.google.com/iam-admin/serviceaccounts"
              newTab
            />
          </>
        }
        disabled={isSubmitting}
      />
      <div className="modal-cta-wrap">
        <Button type="submit" isLoading={isSubmitting} disabled={isSubmitting}>
          {submitBtnText}
        </Button>
        <Button variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default GoogleCASForm;
