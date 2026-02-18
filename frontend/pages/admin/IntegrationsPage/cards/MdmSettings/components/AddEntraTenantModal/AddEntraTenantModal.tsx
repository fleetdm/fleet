import React, { useState, useContext } from "react";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import configAPI from "services/entities/config";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";

import { IAddTenantFormValidation, validateFormData } from "./helpers";

const baseClass = "add-entra-tenant-modal";

export interface IAddTenantFormData {
  tenantId?: string;
}

interface IAddEntraTenantModalProps {
  onExit: () => void;
}

const AddEntraTenantModal = ({ onExit }: IAddEntraTenantModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { setConfig, config } = useContext(AppContext);

  const [isAdding, setIsAdding] = React.useState(false);
  const [formData, setFormData] = React.useState<IAddTenantFormData>({
    tenantId: undefined,
  });
  const [
    formValidation,
    setFormValidation,
  ] = useState<IAddTenantFormValidation>(() =>
    validateFormData({
      tenantId: formData.tenantId,
    })
  );

  const onChangeTenantID = (value: string) => {
    const newFormData = { tenantId: value };
    setFormData(newFormData);
    const newErrs = validateFormData(newFormData);
    setFormValidation(newErrs);
  };

  const onAddTenant = async () => {
    const { tenantId } = formData;

    const validation = validateFormData({ tenantId });

    // do an additional validation to check if the tenant id already exists in the config
    const tenantIdExists =
      config?.mdm.windows_entra_tenant_ids?.includes(tenantId ?? "") ?? false;
    if (tenantIdExists) {
      renderFlash("error", "Couldn't add tenant. Tenant ID already exists.");
      return;
    }

    if (validation.isValid && !tenantIdExists) {
      setIsAdding(true);
      const currentTenantIds = config?.mdm.windows_entra_tenant_ids ?? [];
      try {
        const updateData = await configAPI.update({
          mdm: {
            windows_entra_tenant_ids: [...currentTenantIds, tenantId],
          },
        });
        setConfig(updateData);
        renderFlash("success", "Successfully added tenant");
        onExit();
      } catch (error) {
        renderFlash("error", "Couldn't add tenant. Please try again");
      } finally {
        setIsAdding(false);
      }
    } else {
      setFormValidation(validation);
    }
  };

  return (
    <Modal
      className={baseClass}
      title="Add Entra tenant"
      onExit={onExit}
      isContentDisabled={isAdding}
    >
      <>
        <div>
          <InputField
            label="Tenant ID"
            name="tenant id"
            placeholder="6d8769e6-0f8b-418d-b385-1a53968781c9"
            value={formData.tenantId}
            onChange={onChangeTenantID}
            error={formValidation.tenantId?.message}
            helpText={
              <>
                Find your <b>Tenant ID</b>, on{" "}
                <CustomLink
                  text="Microsoft Entra ID > Home"
                  url="https://fleetdm.com/learn-more-about/microsoft-entra-tenant-id"
                  newTab
                />
              </>
            }
          />
        </div>
        <div className="modal-cta-wrap">
          <Button
            onClick={onAddTenant}
            disabled={!formValidation.isValid}
            isLoading={isAdding}
          >
            Add
          </Button>
          <Button onClick={onExit} variant="inverse">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default AddEntraTenantModal;
