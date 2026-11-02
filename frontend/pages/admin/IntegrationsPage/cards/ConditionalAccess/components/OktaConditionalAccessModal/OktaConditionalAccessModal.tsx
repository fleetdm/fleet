import React, { useCallback, useContext, useState } from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import FileUploader from "components/FileUploader";
import InputField from "components/forms/fields/InputField";
import { isValidURL } from "components/forms/validators";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import TooltipWrapper from "components/TooltipWrapper";
import { AppContext } from "context/app";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import useGitOpsMode from "hooks/useGitOpsMode";
import { IConfig, isOktaConditionalAccessConfigured } from "interfaces/config";
import conditionalAccessAPI from "services/entities/conditional_access";
import configAPI from "services/entities/config";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

const baseClass = "okta-conditional-access-modal";

const OKTA_IDP_ID = "okta_idp_id";
const OKTA_ACS_URL = "okta_acs_url";
const OKTA_AUDIENCE_URI = "okta_audience_uri";
const OKTA_CERTIFICATE = "okta_certificate";

const MAX_URL_LENGTH = 2048;
const MAX_CERT_LENGTH = 8192;

interface IFormData {
  [OKTA_IDP_ID]: string;
  [OKTA_ACS_URL]: string;
  [OKTA_AUDIENCE_URI]: string;
  [OKTA_CERTIFICATE]: string;
}

const validate = (data: IFormData): IFormErrors => {
  const errors: IFormErrors = {};

  if (!data[OKTA_IDP_ID].trim()) {
    errors[OKTA_IDP_ID] = "Enter IdP ID";
  } else if (data[OKTA_IDP_ID].length > MAX_URL_LENGTH) {
    errors[
      OKTA_IDP_ID
    ] = `IdP ID must be ${MAX_URL_LENGTH} characters or fewer`;
  }

  if (!data[OKTA_ACS_URL].trim()) {
    errors[OKTA_ACS_URL] = "Enter assertion consumer service URL";
  } else if (data[OKTA_ACS_URL].length > MAX_URL_LENGTH) {
    errors[
      OKTA_ACS_URL
    ] = `Assertion consumer service URL must be ${MAX_URL_LENGTH} characters or fewer`;
  } else if (
    !isValidURL({ url: data[OKTA_ACS_URL], protocols: ["http", "https"] })
  ) {
    errors[OKTA_ACS_URL] = "Enter a valid URL with http or https scheme";
  }

  if (!data[OKTA_AUDIENCE_URI].trim()) {
    errors[OKTA_AUDIENCE_URI] = "Enter audience URI";
  } else if (data[OKTA_AUDIENCE_URI].length > MAX_URL_LENGTH) {
    errors[
      OKTA_AUDIENCE_URI
    ] = `Audience URI must be ${MAX_URL_LENGTH} characters or fewer`;
  }

  if (!data[OKTA_CERTIFICATE].trim()) {
    errors[OKTA_CERTIFICATE] = "Upload certificate";
  } else if (data[OKTA_CERTIFICATE].length > MAX_CERT_LENGTH) {
    errors[
      OKTA_CERTIFICATE
    ] = `Certificate must be ${MAX_CERT_LENGTH} characters or fewer`;
  }

  return errors;
};

export interface IOktaConditionalAccessModalProps {
  onCancel: () => void;
  onSuccess: (updatedConfig: IConfig) => void;
}

const OktaConditionalAccessModal = ({
  onCancel,
  onSuccess,
}: IOktaConditionalAccessModalProps) => {
  const { config } = useContext(AppContext);
  const { gitOpsModeEnabled } = useGitOpsMode();

  const savedSettings = config?.conditional_access;
  const [certFile, setCertFile] = useState<File | null>(null);
  // The saved certificate has no file name; this tracks whether the
  // uploader should still show it as the current file.
  const [showSavedCert, setShowSavedCert] = useState(
    !!savedSettings?.okta_certificate
  );

  const {
    formData,
    setField,
    commitFields,
    getError,
    clearFieldError,
    validateField,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IFormData>({
    initialFormData: {
      [OKTA_IDP_ID]: savedSettings?.okta_idp_id ?? "",
      [OKTA_ACS_URL]: savedSettings?.okta_assertion_consumer_service_url ?? "",
      [OKTA_AUDIENCE_URI]: savedSettings?.okta_audience_uri ?? "",
      [OKTA_CERTIFICATE]: savedSettings?.okta_certificate ?? "",
    },
    validate,
    // The certificate is PEM content pulled from a file; a trailing newline
    // after -----END CERTIFICATE----- is part of the format.
    skipTrim: [OKTA_CERTIFICATE],
  });

  const [isDownloadingCert, setIsDownloadingCert] = useState(false);

  const onDownloadSigningCert = useCallback(async () => {
    setIsDownloadingCert(true);
    try {
      const blob = await conditionalAccessAPI.getIdpSigningCert();
      const url = URL.createObjectURL(blob);
      const downloadLink = document.createElement("a");
      downloadLink.href = url;
      downloadLink.download = "fleet-idp-signing-cert.pem";
      downloadLink.click();
      downloadLink.remove();
      URL.revokeObjectURL(url);
    } catch (e: unknown) {
      notify.error("Failed to download signing certificate.", { response: e });
    } finally {
      setIsDownloadingCert(false);
    }
  }, []);

  const onValidSubmit = async (data: IFormData) => {
    if (!config) return;
    const isUpdate = isOktaConditionalAccessConfigured(config);
    try {
      const updatedConfig = await configAPI.update({
        conditional_access: {
          okta_idp_id: data[OKTA_IDP_ID],
          okta_assertion_consumer_service_url: data[OKTA_ACS_URL],
          okta_audience_uri: data[OKTA_AUDIENCE_URI],
          okta_certificate: data[OKTA_CERTIFICATE],
          // Preserve existing Microsoft Entra settings
          microsoft_entra_tenant_id:
            config.conditional_access?.microsoft_entra_tenant_id || "",
        },
      });
      notify.success(
        isUpdate
          ? "Successfully updated Okta conditional access"
          : "Successfully configured Okta conditional access"
      );
      onSuccess(updatedConfig);
    } catch (e) {
      notify.error(
        "Could not update conditional access integration settings.",
        { response: e }
      );
    }
  };

  const onDeleteFile = () => {
    setCertFile(null);
    setShowSavedCert(false);
    commitFields({ [OKTA_CERTIFICATE]: "" });
  };

  const onSelectFile = useCallback(
    (files: FileList | null) => {
      const file = files?.[0];
      if (!file) return;

      if (!file.name.match(/\.(pem|crt|cer|cert)$/i)) {
        notify.error(
          "Invalid file type. Please upload a .pem, .crt, .cer, or .cert file."
        );
        return;
      }

      const reader = new FileReader();
      reader.readAsText(file);

      reader.addEventListener("load", () => {
        const content = reader.result as string;

        if (
          !content.includes("-----BEGIN CERTIFICATE-----") ||
          !content.includes("-----END CERTIFICATE-----")
        ) {
          notify.error(
            "Invalid certificate format. The file must be a valid PEM-encoded certificate."
          );
          return;
        }

        setCertFile(file);
        commitFields({ [OKTA_CERTIFICATE]: content });
      });

      reader.addEventListener("error", () => {
        notify.error("Failed to read the certificate file.");
      });
    },
    [commitFields]
  );

  const getCertFileDetails = () => {
    if (certFile) {
      return { name: certFile.name };
    }
    if (showSavedCert) {
      return { name: "Okta certificate" };
    }
    return undefined;
  };

  return (
    <Modal
      title="Okta conditional access"
      onExit={onCancel}
      className={baseClass}
      width="xlarge"
    >
      <form onSubmit={handleSubmit(onValidSubmit)} autoComplete="off">
        <p className={`${baseClass}__instructions`}>
          To configure Okta conditional access, follow the instructions in the{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/okta-conditional-access`}
            text="guide"
            newTab
          />
        </p>

        {/* IdP Signature Certificate Section */}
        <div className={`${baseClass}__certificate-section`}>
          <TooltipWrapper
            tipContent="Upload this certificate in Okta when creating the Fleet IdP."
            underline
          >
            Identity provider (IdP) signature certificate
          </TooltipWrapper>
          <Button
            variant="secondary"
            onClick={onDownloadSigningCert}
            isLoading={isDownloadingCert}
            disabled={isDownloadingCert}
            icon="download"
            iconPosition="right"
          >
            <span>Download certificate</span>
          </Button>
        </div>

        {/* Help text */}
        <p className={`${baseClass}__field-instructions`}>
          You can find the following fields in Okta after creating an IdP in{" "}
          <strong>Security</strong> &gt; <strong>Identity Providers</strong>{" "}
          &gt; <strong>SAML 2.0 IdP</strong>.
        </p>

        <div
          className={`form ${
            gitOpsModeEnabled ? "disabled-by-gitops-mode" : ""
          }`}
        >
          <InputField
            label="IdP ID"
            name={OKTA_IDP_ID}
            value={formData[OKTA_IDP_ID]}
            onChange={(value: string) => setField(OKTA_IDP_ID, value)}
            onFocus={() => clearFieldError(OKTA_IDP_ID)}
            onBlur={() => validateField(OKTA_IDP_ID)}
            error={getError(OKTA_IDP_ID)}
            disabled={isSubmitting || gitOpsModeEnabled}
          />
          <InputField
            label="Assertion consumer service URL"
            name={OKTA_ACS_URL}
            value={formData[OKTA_ACS_URL]}
            onChange={(value: string) => setField(OKTA_ACS_URL, value)}
            onFocus={() => clearFieldError(OKTA_ACS_URL)}
            onBlur={() => validateField(OKTA_ACS_URL)}
            error={getError(OKTA_ACS_URL)}
            disabled={isSubmitting || gitOpsModeEnabled}
          />
          <InputField
            label="Audience URI"
            name={OKTA_AUDIENCE_URI}
            value={formData[OKTA_AUDIENCE_URI]}
            onChange={(value: string) => setField(OKTA_AUDIENCE_URI, value)}
            onFocus={() => clearFieldError(OKTA_AUDIENCE_URI)}
            onBlur={() => validateField(OKTA_AUDIENCE_URI)}
            error={getError(OKTA_AUDIENCE_URI)}
            disabled={isSubmitting || gitOpsModeEnabled}
          />
          <FileUploader
            graphicName="file-pem"
            title="Okta certificate"
            message={
              <>
                Upload the certificate provided by Okta during the{" "}
                <strong>Set Up Authenticator</strong> workflow
              </>
            }
            internalError={getError(OKTA_CERTIFICATE)}
            onFileUpload={onSelectFile}
            buttonType="secondary"
            buttonMessage="Upload"
            accept=".pem,.crt,.cer,.cert"
            fileDetails={getCertFileDetails()}
            onDeleteFile={onDeleteFile}
            disabled={isSubmitting || gitOpsModeEnabled}
          />
        </div>
        <div className="modal-cta-wrap">
          <GitOpsModeTooltipWrapper
            renderChildren={(disableChildren) => (
              <Button
                type="submit"
                disabled={disableChildren || isSubmitting}
                isLoading={isSubmitting}
              >
                Save
              </Button>
            )}
          />
          <Button onClick={onCancel} variant="secondary">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default OktaConditionalAccessModal;
