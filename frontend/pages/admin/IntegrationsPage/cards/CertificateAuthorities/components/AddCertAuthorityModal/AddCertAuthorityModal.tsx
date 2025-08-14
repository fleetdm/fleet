import React, { useContext, useMemo, useState } from "react";

import { NotificationContext } from "context/notification";
import certificatesAPI from "services/entities/certificates";
import { ICertificateAuthorityType } from "interfaces/integration";
import { AppContext } from "context/app";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Modal from "components/Modal";

import { generateDropdownOptions, getErrorMessage } from "./helpers";

import DigicertForm from "../DigicertForm";
import { IDigicertFormData } from "../DigicertForm/DigicertForm";
import { useCertAuthorityDataGenerator } from "../DeleteCertificateAuthorityModal/helpers";
import NDESForm from "../NDESForm";
import { INDESFormData } from "../NDESForm/NDESForm";
import CustomSCEPForm from "../CustomSCEPForm";
import { ICustomSCEPFormData } from "../CustomSCEPForm/CustomSCEPForm";
import HydrantForm from "../HydrantForm";
import { IHydrantFormData } from "../HydrantForm/HydrantForm";

export type ICertFormData =
  | IDigicertFormData
  | IHydrantFormData
  | INDESFormData
  | ICustomSCEPFormData;

const baseClass = "add-cert-authority-modal";

interface IAddCertAuthorityModalProps {
  onExit: () => void;
}

const AddCertAuthorityModal = ({ onExit }: IAddCertAuthorityModalProps) => {
  const { config, setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  const [
    certAuthorityType,
    setCertAuthorityType,
  ] = useState<ICertificateAuthorityType>("digicert");
  const [isAdding, setIsAdding] = useState(false);
  const [digicertFormData, setDigicertFormData] = useState<IDigicertFormData>({
    name: "",
    url: "https://one.digicert.com",
    apiToken: "",
    profileId: "",
    commonName: "",
    userPrincipalName: "",
    certificateSeatId: "",
  });
  const [hydrantFormData, setHydrantFormData] = useState({
    name: "",
    url: "",
    clientId: "",
    clientSecret: "",
  });
  const [ndesFormData, setNDESFormData] = useState<INDESFormData>({
    scepURL: "",
    adminURL: "",
    username: "",
    password: "",
  });
  const [
    customSCEPFormData,
    setCustomSCEPFormData,
  ] = useState<ICustomSCEPFormData>({
    name: "",
    scepURL: "",
    challenge: "",
  });

  const { generateAddPatchData } = useCertAuthorityDataGenerator(
    certAuthorityType
  );

  const onChangeDropdown = (value: ICertificateAuthorityType) => {
    setCertAuthorityType(value);
  };

  const onChangeForm = (update: { name: string; value: string }) => {
    let setFormData;
    let formData: ICertFormData;
    switch (certAuthorityType) {
      case "digicert":
        setFormData = setDigicertFormData;
        formData = digicertFormData;
        break;
      case "hydrant":
        setFormData = setHydrantFormData;
        formData = hydrantFormData;
        break;
      case "ndes":
        setFormData = setNDESFormData;
        formData = ndesFormData;
        break;
      case "custom":
        setFormData = setCustomSCEPFormData;
        formData = customSCEPFormData;
        break;
      default:
        return;
    }

    (setFormData as React.Dispatch<React.SetStateAction<ICertFormData>>)({
      ...formData,
      [update.name]: update.value,
    });
  };

  const onAddCertAuthority = async () => {
    let formData: ICertFormData;
    switch (certAuthorityType) {
      case "digicert":
        formData = digicertFormData;
        break;
      case "hydrant":
        formData = hydrantFormData;
        break;
      case "ndes":
        formData = ndesFormData;
        break;
      case "custom":
        formData = customSCEPFormData;
        break;
      default:
        return;
    }

    const addPatchData = generateAddPatchData(formData);
    setIsAdding(true);
    try {
      const newConfig = await certificatesAPI.addCertificateAuthority(
        addPatchData
      );
      renderFlash("success", "Successfully added your certificate authority.");
      onExit();
      setConfig(newConfig);
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    }
    setIsAdding(false);
  };

  const dropdownOptions = useMemo(() => {
    return generateDropdownOptions(!!config?.integrations.ndes_scep_proxy);
  }, [config?.integrations.ndes_scep_proxy]);

  const renderForm = () => {
    const submitBtnText = "Add CA";

    switch (certAuthorityType) {
      case "digicert":
        return (
          <DigicertForm
            formData={digicertFormData}
            submitBtnText={submitBtnText}
            isSubmitting={isAdding}
            onChange={onChangeForm}
            onSubmit={onAddCertAuthority}
            onCancel={onExit}
          />
        );
      case "hydrant":
        return (
          <HydrantForm
            formData={hydrantFormData}
            submitBtnText={submitBtnText}
            isSubmitting={isAdding}
            onChange={onChangeForm}
            onSubmit={onAddCertAuthority}
            onCancel={onExit}
          />
        );
      case "ndes":
        return (
          <NDESForm
            formData={ndesFormData}
            submitBtnText={submitBtnText}
            isSubmitting={isAdding}
            onChange={onChangeForm}
            onSubmit={onAddCertAuthority}
            onCancel={onExit}
          />
        );
      case "custom":
        return (
          <CustomSCEPForm
            formData={customSCEPFormData}
            submitBtnText={submitBtnText}
            isSubmitting={isAdding}
            onChange={onChangeForm}
            onSubmit={onAddCertAuthority}
            onCancel={onExit}
          />
        );
      default:
        return null;
    }
  };

  return (
    <Modal
      className={baseClass}
      title="Add certificate authority (CA)"
      width="large"
      onExit={onExit}
      isContentDisabled={isAdding}
    >
      <>
        <Dropdown
          options={dropdownOptions}
          value={certAuthorityType}
          className={`${baseClass}__cert-authority-dropdown`}
          onChange={onChangeDropdown}
          searchable={false}
        />
        {renderForm()}
      </>
    </Modal>
  );
};

export default AddCertAuthorityModal;
