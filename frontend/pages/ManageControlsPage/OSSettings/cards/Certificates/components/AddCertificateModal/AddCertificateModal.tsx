import React, { useContext, useMemo, useState } from "react";
import { useQuery } from "react-query";
import { SingleValue } from "react-select-5";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import paths from "router/paths";

import { NotificationContext } from "context/notification";
import certificatesAPI, { ICertificate } from "services/entities/certificates";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import TooltipWrapper from "components/TooltipWrapper";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import CustomLink from "components/CustomLink";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";

import {
  validateFormData,
  generateFormValidations,
  IAddCertFormValidation,
} from "./helpers";

const baseClass = "add-ct-modal";

export interface IAddCertFormData {
  name: string;
  certAuthorityId: string;
  subjectName: string;
}

interface IAddCertModalProps {
  existingCerts: ICertificate[];
  onExit: () => void;
  onSuccess: () => void;
  currentTeamId?: number;
}

const AddCertModal = ({
  existingCerts: existingCTs,
  onExit,
  onSuccess,
  currentTeamId,
}: IAddCertModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [isUpdating, setIsUpdating] = useState(false);
  const [formData, setFormData] = useState<IAddCertFormData>({
    name: "",
    certAuthorityId: "",
    subjectName: "",
  });

  const validations = useMemo(
    () => generateFormValidations(existingCTs || []),
    [existingCTs]
  );

  const [formValidation, setFormValidation] = useState<IAddCertFormValidation>(
    () => validateFormData(formData, validations)
  );

  const {
    data: cAResp,
    isLoading: isLoadingCAs,
    isError: isErrorCAs,
  } = useQuery(
    "certAuthorities",
    () => {
      return certificatesAPI.getCertificateAuthoritiesList();
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (data) => data.certificate_authorities,
    }
  );
  const caPartials = cAResp ?? [];

  const caDropdownOptions = caPartials.map((cAP) => ({
    value: cAP.id.toString(),
    label: cAP.name,
  }));

  const onInputChange = (update: { name: string; value: string }) => {
    const updatedFormData = { ...formData, [update.name]: update.value };
    setFormData(updatedFormData);
    setFormValidation(validateFormData(updatedFormData, validations));
  };

  const onChangeCA = (newValue: SingleValue<CustomOptionType>) => {
    const updatedFormData = {
      ...formData,
      certAuthorityId: newValue?.value ?? "",
    };
    setFormData(updatedFormData);
    setFormValidation(validateFormData(updatedFormData, validations));
  };

  const onSubmitForm = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    setIsUpdating(true);
    try {
      await certificatesAPI.createCert({
        name: formData.name,
        certAuthorityId: parseInt(formData.certAuthorityId, 10),
        subjectName: formData.subjectName,
        teamId: currentTeamId,
      });
      renderFlash("success", "Successfully added your certificate.");
      onSuccess();
      onExit();
    } catch (e) {
      renderFlash("error", "Couldn't add certificate. Please try again.");
    } finally {
      setIsUpdating(false);
    }
  };

  const renderForm = () => {
    if (isLoadingCAs) {
      return <Spinner />;
    }

    if (isErrorCAs) {
      return <DataError />;
    }
    return (
      <form className={baseClass} onSubmit={onSubmitForm}>
        <InputField
          name="name"
          label="Name"
          value={formData.name}
          onChange={onInputChange}
          error={formValidation.name?.message}
          helpText="Letters, numbers, spaces, dashes, and underscores only. Name can be used as certificate alias to reference in configuration profiles."
          parseTarget
          placeholder="VPN certificate"
        />
        <DropdownWrapper
          label="Certificate authority (CA)"
          name="certificateAuthority"
          options={caDropdownOptions}
          value={formData.certAuthorityId}
          onChange={onChangeCA}
          customNoOptionsMessage="No certificate authorities found."
          placeholder="Select certificate authority"
          helpText={
            <>
              Certificate will be issued from this CA. Currently, only custom
              SCEP CA is supported. You can add CAs on the
              <CustomLink
                url={paths.ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES}
                text="Certificate authorities"
              />{" "}
              page.
            </>
          }
          error={formValidation.certAuthorityId?.message}
        />
        <InputField
          name="subjectName"
          label="Subject name (SN)"
          type="textarea"
          value={formData.subjectName}
          onChange={onInputChange}
          error={formValidation.subjectName?.message}
          helpText='Separate subject fields by ", ". For example: CN=john@example.com, O=Acme Inc.'
          parseTarget
          placeholder="CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME, O=Your Organization"
        />
        <div className="modal-cta-wrap">
          <TooltipWrapper
            tipContent="Complete all required fields to save"
            underline={false}
            position="top"
            disableTooltip={formValidation.isValid}
            showArrow
          >
            <Button
              isLoading={isUpdating}
              disabled={!formValidation.isValid || isUpdating}
              type="submit"
            >
              Create
            </Button>
          </TooltipWrapper>
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </form>
    );
  };

  return (
    <Modal
      className={baseClass}
      title="Add certificate"
      width="large"
      onExit={onExit}
      isContentDisabled={isUpdating}
    >
      {renderForm()}
    </Modal>
  );
};

export default AddCertModal;
