import React, { useContext, useMemo, useState } from "react";

import { NotificationContext } from "context/notification";
import certificatesAPI from "services/entities/certificates";
import {
  ICertificateAuthorityPartial,
  ICertificateAuthorityType,
} from "interfaces/certificates";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Modal from "components/Modal";

import {
  generateAddCertAuthorityData,
  generateDropdownOptions,
  getErrorMessage,
} from "./helpers";

import DigicertForm from "../DigicertForm";
import { IDigicertFormData } from "../DigicertForm/DigicertForm";
import NDESForm from "../NDESForm";
import { INDESFormData } from "../NDESForm/NDESForm";
import CustomSCEPForm from "../CustomSCEPForm";
import { ICustomSCEPFormData } from "../CustomSCEPForm/CustomSCEPForm";
import HydrantForm from "../HydrantForm";
import { IHydrantFormData } from "../HydrantForm/HydrantForm";
import SmallstepForm, {
  ISmallstepFormData,
} from "../SmallstepForm/SmallstepForm";

export type ICertFormData =
  | IDigicertFormData
  | IHydrantFormData
  | INDESFormData
  | ICustomSCEPFormData
  | ISmallstepFormData;

const baseClass = "add-cert-authority-modal";

interface IAddCertAuthorityModalProps {
  certAuthorities: ICertificateAuthorityPartial[];
  onExit: () => void;
}

const AddCertAuthorityModal = ({
  certAuthorities,
  onExit,
}: IAddCertAuthorityModalProps) => {
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
  const [
    smallstepFormData,
    setSmallstepFormData,
  ] = useState<ISmallstepFormData>({
    name: "",
    scepURL: "",
    challengeURL: "",
    username: "",
    password: "",
  });

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
      case "ndes_scep_proxy":
        setFormData = setNDESFormData;
        formData = ndesFormData;
        break;
      case "custom_scep_proxy":
        setFormData = setCustomSCEPFormData;
        formData = customSCEPFormData;
        break;
      case "smallstep":
        setFormData = setSmallstepFormData;
        formData = smallstepFormData;
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
      case "ndes_scep_proxy":
        formData = ndesFormData;
        break;
      case "custom_scep_proxy":
        formData = customSCEPFormData;
        break;
      case "smallstep":
        formData = smallstepFormData;
        break;
      default:
        return;
    }

    const addCertAuthorityData = generateAddCertAuthorityData(
      certAuthorityType,
      formData
    );
    if (!addCertAuthorityData) {
      return;
    }
    setIsAdding(true);
    try {
      await certificatesAPI.addCertificateAuthority(addCertAuthorityData);
      renderFlash("success", "Successfully added your certificate authority.");
      onExit();
    } catch (e) {
      renderFlash("error", getErrorMessage(e));
    }
    setIsAdding(false);
  };

  const dropdownOptions = useMemo(() => {
    return generateDropdownOptions(
      certAuthorities.some((cert) => cert.type === "ndes_scep_proxy")
    );
  }, [certAuthorities]);

  const renderForm = () => {
    const submitBtnText = "Add CA";

    switch (certAuthorityType) {
      case "digicert":
        return (
          <DigicertForm
            formData={digicertFormData}
            certAuthorities={certAuthorities}
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
            certAuthorities={certAuthorities}
            submitBtnText={submitBtnText}
            isSubmitting={isAdding}
            onChange={onChangeForm}
            onSubmit={onAddCertAuthority}
            onCancel={onExit}
          />
        );
      case "ndes_scep_proxy":
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
      case "custom_scep_proxy":
        return (
          <CustomSCEPForm
            formData={customSCEPFormData}
            certAuthorities={certAuthorities}
            submitBtnText={submitBtnText}
            isSubmitting={isAdding}
            onChange={onChangeForm}
            onSubmit={onAddCertAuthority}
            onCancel={onExit}
          />
        );
      case "smallstep":
        return (
          <SmallstepForm
            formData={smallstepFormData}
            certAuthorities={certAuthorities}
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
