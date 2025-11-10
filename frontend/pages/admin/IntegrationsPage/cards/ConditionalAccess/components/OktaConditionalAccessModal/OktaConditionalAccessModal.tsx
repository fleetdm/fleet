import React, { useCallback, useContext, useState } from "react";
import { size } from "lodash";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import configAPI from "services/entities/config";
import { IConfig } from "interfaces/config";
import endpoints from "utilities/endpoints";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import CustomLink from "components/CustomLink";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import { IInputFieldParseTarget } from "interfaces/form_field";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import FileUploader from "components/FileUploader";

const baseClass = "okta-conditional-access-modal";

const OKTA_IDP_ID = "okta_idp_id";
const OKTA_ACS_URL = "okta_assertion_consumer_service_url";
const OKTA_AUDIENCE_URI = "okta_audience_uri";
const OKTA_CERTIFICATE = "okta_certificate";

interface IFormData {
  [OKTA_IDP_ID]: string;
  [OKTA_ACS_URL]: string;
  [OKTA_AUDIENCE_URI]: string;
  [OKTA_CERTIFICATE]: string;
}

interface IFormErrors {
  [OKTA_IDP_ID]?: string | null;
  [OKTA_ACS_URL]?: string | null;
  [OKTA_AUDIENCE_URI]?: string | null;
  [OKTA_CERTIFICATE]?: string | null;
}

const validate = (formData: IFormData) => {
  const errs: IFormErrors = {};
  if (!formData[OKTA_IDP_ID]) {
    errs[OKTA_IDP_ID] = "IdP ID must be present";
  }
  if (!formData[OKTA_ACS_URL]) {
    errs[OKTA_ACS_URL] = "Assertion Consumer Service URL must be present";
  }
  if (!formData[OKTA_AUDIENCE_URI]) {
    errs[OKTA_AUDIENCE_URI] = "Audience URI must be present";
  }
  if (!formData[OKTA_CERTIFICATE]) {
    errs[OKTA_CERTIFICATE] = "Certificate must be present";
  }
  return errs;
};

export interface IOktaConditionalAccessModalProps {
  onCancel: () => void;
  onSuccess: (updatedConfig: IConfig) => void;
}

const OktaConditionalAccessModal = ({
  onCancel,
  onSuccess,
}: IOktaConditionalAccessModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { config } = useContext(AppContext);

  const [isUpdating, setIsUpdating] = useState(false);
  const [formData, setFormData] = useState<IFormData>({
    [OKTA_IDP_ID]: "",
    [OKTA_ACS_URL]: "",
    [OKTA_AUDIENCE_URI]: "",
    [OKTA_CERTIFICATE]: "",
  });
  const [formErrors, setFormErrors] = useState<IFormErrors>({});
  const [certFile, setCertFile] = useState<File | null>(null);

  const onSubmit = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const errs = validate(formData);
    if (Object.keys(errs).length > 0) {
      setFormErrors(errs);
      return;
    }
    setIsUpdating(true);
    try {
      const updatedConfig = await configAPI.update({
        conditional_access: {
          okta_idp_id: formData[OKTA_IDP_ID],
          okta_assertion_consumer_service_url: formData[OKTA_ACS_URL],
          okta_audience_uri: formData[OKTA_AUDIENCE_URI],
          okta_certificate: formData[OKTA_CERTIFICATE],
          // Preserve existing Microsoft Entra settings
          microsoft_entra_tenant_id:
            config?.conditional_access?.microsoft_entra_tenant_id || "",
        },
      });
      renderFlash("success", "Successfully configured Okta conditional access");
      setIsUpdating(false);
      onSuccess(updatedConfig);
    } catch (e) {
      renderFlash(
        "error",
        "Could not update conditional access integration settings."
      );
      setIsUpdating(false);
    }
  };

  const onInputChange = ({ name, value }: IInputFieldParseTarget) => {
    const newFormData = { ...formData, [name]: value };
    setFormData(newFormData);
    const newErrs = validate(newFormData);
    // only set errors that are updates of existing errors
    // new errors are only set onBlur or submit
    const errsToSet: Record<string, string> = {};
    Object.keys(formErrors).forEach((k) => {
      // @ts-ignore
      if (newErrs[k]) {
        // @ts-ignore
        errsToSet[k] = newErrs[k];
      }
    });
    setFormErrors(errsToSet);
  };

  const onInputBlur = () => {
    setFormErrors(validate(formData));
  };

  const onDeleteFile = () => {
    setCertFile(null);
    setFormData({ ...formData, [OKTA_CERTIFICATE]: "" });
    setFormErrors({
      ...formErrors,
      [OKTA_CERTIFICATE]: "Certificate must be present",
    });
  };

  const onSelectFile = useCallback(
    (files: FileList | null) => {
      const file = files?.[0];
      if (!file) return;

      // Validate file extension
      if (!file.name.match(/\.(pem|crt|cer|cert)$/i)) {
        renderFlash(
          "error",
          "Invalid file type. Please upload a .pem, .crt, .cer, or .cert file."
        );
        return;
      }

      const reader = new FileReader();
      reader.readAsText(file);

      reader.addEventListener("load", () => {
        const content = reader.result as string;

        // Validate PEM format
        if (
          !content.includes("-----BEGIN CERTIFICATE-----") ||
          !content.includes("-----END CERTIFICATE-----")
        ) {
          renderFlash(
            "error",
            "Invalid certificate format. The file must be a valid PEM-encoded certificate."
          );
          return;
        }

        // Create new form data with the certificate
        const newFormData = { ...formData, [OKTA_CERTIFICATE]: content };

        // Store the certificate content and file details
        setCertFile(file);
        setFormData(newFormData);
        // Re-validate the entire form to clear errors if all fields are now complete
        setFormErrors(validate(newFormData));
      });

      reader.addEventListener("error", () => {
        renderFlash("error", "Failed to read the certificate file.");
      });
    },
    [formData, renderFlash]
  );

  return (
    <Modal
      title="Okta conditional access"
      onExit={onCancel}
      className={baseClass}
      width="xlarge"
    >
      <>
        <form onSubmit={onSubmit} autoComplete="off">
          <p className={`${baseClass}__instructions`}>
            To configure Okta conditional access, follow the instructions in the{" "}
            <CustomLink
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/okta-setup`}
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
            <br />
            <a
              href={endpoints.CONDITIONAL_ACCESS_IDP_SIGNING_CERT}
              download="fleet-idp-signing-certificate.pem"
              className="button button--inverse"
            >
              Download certificate&nbsp;&nbsp;
              <Icon name="download" />
            </a>
          </div>

          {/* User Scope Profile */}
          <InputField
            enableCopy
            label="User scope profile"
            readOnly
            value="TODO read-only profile goes here"
            type="textarea"
          />

          {/* Help text */}
          <p className={`${baseClass}__field-instructions`}>
            You can find the following fields in Okta after creating an IdP in{" "}
            <strong>Security</strong> &gt; <strong>Identity Providers</strong>{" "}
            &gt; <strong>SAML 2.0 IdP</strong>.
          </p>

          <InputField
            label="IdP ID"
            onChange={onInputChange}
            name={OKTA_IDP_ID}
            value={formData[OKTA_IDP_ID]}
            parseTarget
            onBlur={onInputBlur}
            error={formErrors[OKTA_IDP_ID]}
          />
          <InputField
            label="Assertion consumer service URL"
            onChange={onInputChange}
            name={OKTA_ACS_URL}
            value={formData[OKTA_ACS_URL]}
            parseTarget
            onBlur={onInputBlur}
            error={formErrors[OKTA_ACS_URL]}
          />
          <InputField
            label="Audience URI"
            onChange={onInputChange}
            name={OKTA_AUDIENCE_URI}
            value={formData[OKTA_AUDIENCE_URI]}
            parseTarget
            onBlur={onInputBlur}
            error={formErrors[OKTA_AUDIENCE_URI]}
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
            onFileUpload={onSelectFile}
            buttonType="brand-inverse-icon"
            buttonMessage="Upload"
            accept=".pem,.crt,.cer,.cert"
            fileDetails={certFile ? { name: certFile.name } : undefined}
            onDeleteFile={onDeleteFile}
          />

          <div className="modal-cta-wrap">
            <Button
              type="submit"
              disabled={!!size(formErrors)}
              isLoading={isUpdating}
            >
              Save
            </Button>
            <Button onClick={onCancel} variant="inverse">
              Cancel
            </Button>
          </div>
        </form>
      </>
    </Modal>
  );
};

export default OktaConditionalAccessModal;
