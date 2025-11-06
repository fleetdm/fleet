import React, { useContext, useState } from "react";
import { size } from "lodash";

import { NotificationContext } from "context/notification";
import conditionalAccessAPI from "services/entities/conditional_access";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import CustomLink from "components/CustomLink";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { IInputFieldParseTarget } from "interfaces/form_field";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

const baseClass = "entra-conditional-access-modal";

const MSETID = "microsoft_entra_tenant_id";

interface IFormData {
  [MSETID]: string;
}

interface IFormErrors {
  [MSETID]?: string | null;
}

const validate = (formData: IFormData) => {
  const errs: IFormErrors = {};
  if (!formData[MSETID]) {
    errs[MSETID] = "Tenant ID must be present";
  }
  return errs;
};

export interface IEntraConditionalAccessModalProps {
  onCancel: () => void;
  onSuccess: () => void;
}

const EntraConditionalAccessModal = ({
  onCancel,
  onSuccess,
}: IEntraConditionalAccessModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [isUpdating, setIsUpdating] = useState(false);
  const [formData, setFormData] = useState<IFormData>({
    [MSETID]: "",
  });
  const [formErrors, setFormErrors] = useState<IFormErrors>({});

  const onSubmit = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const errs = validate(formData);
    if (Object.keys(errs).length > 0) {
      setFormErrors(errs);
      return;
    }
    setIsUpdating(true);
    try {
      const {
        microsoft_authentication_url: msAuthURL,
      } = await conditionalAccessAPI.triggerMicrosoftConditionalAccess(
        formData[MSETID]
      );
      window.open(msAuthURL);
      setIsUpdating(false);
      // Close modal and show banner on main page
      onSuccess();
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

  return (
    <Modal
      title="Microsoft Entra conditional access"
      onExit={onCancel}
      className={baseClass}
      width="large"
    >
      <>
        <form onSubmit={onSubmit} autoComplete="off">
          <p className={`${baseClass}__instructions`}>
            To configure Microsoft Entra conditional access, follow the
            instructions in the{" "}
            <CustomLink
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/microsoft-entra-setup`}
              text="guide"
              newTab
            />
          </p>
          <InputField
            label="Microsoft Entra tenant ID"
            helpText="You can find this in your Microsoft Entra admin center."
            onChange={onInputChange}
            name={MSETID}
            value={formData[MSETID]}
            parseTarget
            onBlur={onInputBlur}
            error={formErrors[MSETID]}
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

export default EntraConditionalAccessModal;
