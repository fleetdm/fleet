import React, { useCallback, useRef, useState } from "react";

import Button from "components/buttons/Button";
import FileUploader from "components/FileUploader";
import InputField from "components/forms/fields/InputField";
import { notify } from "components/ToastNotification";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import { ICertificateAuthorityPartial } from "interfaces/certificates";
import { MAX_ENTITY_CHAR_LENGTH } from "utilities/constants";
import { monthDayYearFormat } from "utilities/date_format";

import {
  MAX_P12_SIZE_BYTES,
  readFileAsBase64,
  validateEJBCAForm,
} from "./helpers";

const baseClass = "ejbca-form";

export interface IEJBCAFormData {
  name: string;
  url: string;
  /** The client certificate and key, as a base64 PKCS#12 */
  clientP12: string;
  password: string;
  trustCABundle: string;
  caName: string;
  certificateProfileName: string;
  endEntityProfileName: string;
  commonName: string;
  usernameTemplate: string;
  userPrincipalName: string;
}

export const EMPTY_EJBCA_FORM_DATA: IEJBCAFormData = {
  name: "",
  url: "",
  clientP12: "",
  password: "",
  trustCABundle: "",
  caName: "",
  certificateProfileName: "",
  endEntityProfileName: "",
  commonName: "",
  usernameTemplate: "",
  userPrincipalName: "",
};

interface IEJBCAFormProps {
  certAuthorities?: ICertificateAuthorityPartial[];
  initialFormData?: IEJBCAFormData;
  /** The id of the CA being edited. Without it, the form adds a CA. */
  editingCertAuthorityId?: number;
  /** When the stored client certificate of the CA being edited expires */
  clientCertExpiresAt?: string;
  submitBtnText: string;
  isSubmitting?: boolean;
  /** The errors for the fields that the server reported on the last submission */
  serverErrors?: IFormErrors | null;
  onSubmit: (formData: IEJBCAFormData) => void | Promise<unknown>;
  onCancel: () => void;
}

const EJBCAForm = ({
  certAuthorities = [],
  initialFormData = EMPTY_EJBCA_FORM_DATA,
  editingCertAuthorityId,
  clientCertExpiresAt,
  submitBtnText,
  isSubmitting: isSubmittingExternal = false,
  serverErrors,
  onSubmit,
  onCancel,
}: IEJBCAFormProps) => {
  const isEditing = editingCertAuthorityId !== undefined;
  const [p12File, setP12File] = useState<File | null>(null);
  // The number of the latest selection of a file, so that the read of an earlier one can't replace it.
  const latestP12Selection = useRef(0);

  const {
    getTextFieldProps,
    getError,
    commitFields,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IEJBCAFormData>({
    initialFormData,
    validate: (data) =>
      validateEJBCAForm(data, {
        certAuthorities,
        editingId: editingCertAuthorityId,
      }),
    serverErrors,
    isSubmitting: isSubmittingExternal,
    // These are sent as they are: a password can have spaces at its ends.
    skipTrim: ["clientP12", "password"],
  });

  const onSelectP12 = useCallback(
    async (files: FileList | null) => {
      const file = files?.[0];
      if (!file) return;
      if (!/\.(p12|pfx)$/i.test(file.name)) {
        notify.error("Invalid file type. Please upload a .p12 or .pfx file.");
        return;
      }
      if (file.size === 0) {
        notify.error("The file is empty. Please upload a .p12 or .pfx file.");
        return;
      }
      if (file.size > MAX_P12_SIZE_BYTES) {
        notify.error("The file is too large. Please upload a file under 1 MB.");
        return;
      }

      // a slower read of an earlier file pick must not replace this one
      latestP12Selection.current += 1;
      const selection = latestP12Selection.current;
      try {
        const clientP12 = await readFileAsBase64(file);
        if (selection === latestP12Selection.current) {
          setP12File(file);
          commitFields({ clientP12 });
        }
      } catch {
        notify.error("Couldn't read the file. Please try again.");
      }
    },
    [commitFields]
  );

  const onDeleteP12 = () => {
    // allow deleting the file even if its read is still in progress
    latestP12Selection.current += 1;
    setP12File(null);
    commitFields({ clientP12: "", password: "" });
  };

  const clientCertInfo =
    isEditing && clientCertExpiresAt
      ? `The client certificate expires on ${monthDayYearFormat(
          clientCertExpiresAt
        )}. Upload a new one to replace it.`
      : "Fleet uses it to authenticate to EJBCA.";

  return (
    <form className={baseClass} onSubmit={handleSubmit(onSubmit)}>
      <InputField
        label="Name"
        {...getTextFieldProps("name")}
        helpText="Letters, numbers, and underscores only. Fleet will create configuration profile variables with the name as suffix (e.g. $FLEET_VAR_EJBCA_DATA_WIFI_CERTIFICATE)."
        placeholder="WIFI_CERTIFICATE"
        disabled={isSubmitting}
        inputOptions={{ maxLength: MAX_ENTITY_CHAR_LENGTH }}
      />
      <InputField
        label="URL"
        {...getTextFieldProps("url")}
        helpText="EJBCA instance URL."
        placeholder="https://ejbca.example.com:8443"
        disabled={isSubmitting}
      />
      <FileUploader
        label={isEditing ? "New client certificate" : "Client certificate"}
        graphicName="file-certificate"
        message="Client certificate and key (.p12)"
        additionalInfo={clientCertInfo}
        accept=".p12,.pfx"
        buttonType="secondary"
        internalError={getError("clientP12")}
        // the error is shown in the details once a file is selected, as the uploader only shows it before
        fileDetails={
          p12File
            ? { name: p12File.name, description: getError("clientP12") }
            : undefined
        }
        onFileUpload={onSelectP12}
        onDeleteFile={onDeleteP12}
        disabled={isSubmitting}
      />
      {(!isEditing || p12File) && (
        <InputField
          label="Client certificate password"
          {...getTextFieldProps("password")}
          type="password"
          helpText="Password of the .p12 file."
          disabled={isSubmitting}
        />
      )}
      <InputField
        label="Trust CA bundle"
        {...getTextFieldProps("trustCABundle")}
        type="textarea"
        inputClassName={`${baseClass}__trust-ca-bundle`}
        helpText="PEM-encoded CA certificates that sign the HTTPS certificate of your EJBCA server. If you leave it empty, Fleet trusts public CAs. (optional)"
        placeholder="-----BEGIN CERTIFICATE-----"
        disabled={isSubmitting}
      />
      <InputField
        label="CA name"
        {...getTextFieldProps("caName")}
        helpText="Name of the CA in EJBCA that issues the certificates."
        placeholder="ManagementCA"
        disabled={isSubmitting}
        inputOptions={{ maxLength: MAX_ENTITY_CHAR_LENGTH }}
      />
      <InputField
        label="Certificate profile name"
        {...getTextFieldProps("certificateProfileName")}
        helpText="Name of the certificate profile in EJBCA."
        placeholder="ENDUSER"
        disabled={isSubmitting}
        inputOptions={{ maxLength: MAX_ENTITY_CHAR_LENGTH }}
      />
      <InputField
        label="End entity profile name"
        {...getTextFieldProps("endEntityProfileName")}
        helpText="Name of the end entity profile in EJBCA."
        placeholder="WIFI_DEVICES"
        disabled={isSubmitting}
        inputOptions={{ maxLength: MAX_ENTITY_CHAR_LENGTH }}
      />
      <InputField
        label="Certificate common name (CN)"
        {...getTextFieldProps("commonName")}
        helpText="Certificates delivered to your hosts will have this CN in the subject."
        placeholder="$FLEET_VAR_HOST_HARDWARE_SERIAL"
        disabled={isSubmitting}
        inputOptions={{ maxLength: MAX_ENTITY_CHAR_LENGTH }}
      />
      <InputField
        label="End entity username"
        {...getTextFieldProps("usernameTemplate")}
        helpText="EJBCA creates an end entity with this username for each host. If you leave it empty, Fleet uses the CN. (optional)"
        placeholder="$FLEET_VAR_HOST_HARDWARE_SERIAL"
        disabled={isSubmitting}
        inputOptions={{ maxLength: MAX_ENTITY_CHAR_LENGTH }}
      />
      <InputField
        label="User principal name (UPN)"
        {...getTextFieldProps("userPrincipalName")}
        helpText="Certificates delivered to your hosts will have this UPN attribute in Subject Alternative Name (SAN). (optional)"
        placeholder="$FLEET_VAR_HOST_HARDWARE_SERIAL"
        disabled={isSubmitting}
        inputOptions={{ maxLength: MAX_ENTITY_CHAR_LENGTH }}
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

export default EJBCAForm;
