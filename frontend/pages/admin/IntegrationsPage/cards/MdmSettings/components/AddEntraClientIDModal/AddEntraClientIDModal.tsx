import React, { useState, useContext } from "react";

import { AppContext } from "context/app";
import configAPI from "services/entities/config";

import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import { notify } from "components/ToastNotification";

import { IAddClientIdFormValidation, validateFormData } from "./helpers";

const baseClass = "add-entra-client-id-modal";

export interface IAddClientIdFormData {
  clientId?: string;
}

interface IAddEntraClientIdModalProps {
  onExit: () => void;
}

const AddEntraClientIdModal = ({ onExit }: IAddEntraClientIdModalProps) => {
  const { setConfig, config } = useContext(AppContext);

  const [isAdding, setIsAdding] = React.useState(false);
  const [formData, setFormData] = React.useState<IAddClientIdFormData>({
    clientId: undefined,
  });
  const [
    formValidation,
    setFormValidation,
  ] = useState<IAddClientIdFormValidation>(() =>
    validateFormData({
      clientId: formData.clientId,
    })
  );

  const onChangeClientId = (value: string) => {
    const newFormData = { clientId: value };
    setFormData(newFormData);
    const newErrs = validateFormData(newFormData);
    setFormValidation(newErrs);
  };

  const onAddClientId = async () => {
    // Normalize to a canonical lower-case GUID before validating, de-duplicating, and sending.
    const clientId = formData.clientId?.trim().toLowerCase();

    const validation = validateFormData({ clientId });

    // do an additional validation to check if the client id already exists in the config
    const clientIdExists =
      config?.mdm.windows_entra_client_ids?.some(
        (id) => id.toLowerCase() === clientId
      ) ?? false;
    if (clientIdExists) {
      notify.error("Couldn't add client ID. Client ID already exists.");
      return;
    }

    if (validation.isValid && !clientIdExists) {
      setIsAdding(true);
      const currentClientIds = config?.mdm.windows_entra_client_ids ?? [];
      try {
        const updateData = await configAPI.update({
          mdm: {
            windows_entra_client_ids: [...currentClientIds, clientId],
          },
        });
        setConfig(updateData);
        notify.success("Successfully added client ID");
        onExit();
      } catch (error) {
        notify.error("Couldn't add client ID. Please try again", {
          response: error,
        });
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
      title="Add Entra client ID"
      onExit={onExit}
      isContentDisabled={isAdding}
    >
      <div>
        <InputField
          label="Client ID"
          name="client id"
          placeholder="6d8769e6-0f8b-418d-b385-1a53968781c9"
          value={formData.clientId}
          onChange={onChangeClientId}
          error={formValidation.clientId?.message}
          helpText={
            <>
              Find your <b>Application (client) ID</b> on{" "}
              <CustomLink
                text="Microsoft Entra ID"
                url="https://fleetdm.com/learn-more-about/microsoft-entra-tenant-id"
                newTab
              />{" "}
              &gt; App registrations &gt; your MDM application &gt; Overview.
            </>
          }
        />
      </div>
      <div className="modal-cta-wrap">
        <Button
          onClick={onAddClientId}
          disabled={!formValidation.isValid}
          isLoading={isAdding}
        >
          Add
        </Button>
        <Button onClick={onExit} variant="secondary">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default AddEntraClientIdModal;
