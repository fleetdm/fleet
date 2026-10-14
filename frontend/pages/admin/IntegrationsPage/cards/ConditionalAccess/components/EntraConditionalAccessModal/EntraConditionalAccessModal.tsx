import React from "react";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import useFormValidation, { IFormErrors } from "hooks/useFormValidation";
import conditionalAccessAPI from "services/entities/conditional_access";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

const baseClass = "entra-conditional-access-modal";

const MSETID = "microsoft_entra_tenant_id";

interface IFormData {
  [MSETID]: string;
}

const validate = (data: IFormData): IFormErrors => {
  const errors: IFormErrors = {};
  if (!data[MSETID].trim()) {
    errors[MSETID] = "Enter tenant ID";
  }
  return errors;
};

export interface IEntraConditionalAccessModalProps {
  onCancel: () => void;
  onSuccess: () => void;
}

const EntraConditionalAccessModal = ({
  onCancel,
  onSuccess,
}: IEntraConditionalAccessModalProps) => {
  const {
    formData,
    setField,
    getError,
    clearFieldError,
    validateField,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IFormData>({
    initialFormData: { [MSETID]: "" },
    validate,
  });

  const onValidSubmit = async (data: IFormData) => {
    try {
      const {
        microsoft_authentication_url: msAuthURL,
      } = await conditionalAccessAPI.triggerMicrosoftConditionalAccess(
        data[MSETID]
      );
      window.open(msAuthURL);
      onSuccess();
    } catch (e) {
      notify.error(
        "Could not update conditional access integration settings.",
        { response: e }
      );
    }
  };

  return (
    <Modal
      title="Microsoft Entra conditional access"
      onExit={onCancel}
      className={baseClass}
      width="large"
    >
      <form onSubmit={handleSubmit(onValidSubmit)} autoComplete="off">
        <p className={`${baseClass}__instructions`}>
          To configure Microsoft Entra conditional access, follow the
          instructions in the{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/entra-conditional-access`}
            text="guide"
            newTab
          />
        </p>
        <InputField
          label="Microsoft Entra tenant ID"
          helpText="You can find this in your Microsoft Entra admin center."
          name={MSETID}
          value={formData[MSETID]}
          onChange={(value: string) => setField(MSETID, value)}
          onFocus={() => clearFieldError(MSETID)}
          onBlur={() => validateField(MSETID)}
          error={getError(MSETID)}
          disabled={isSubmitting}
        />
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            disabled={isSubmitting}
            isLoading={isSubmitting}
          >
            Save
          </Button>
          <Button onClick={onCancel} variant="secondary">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default EntraConditionalAccessModal;
