import React, { useMemo, useState } from "react";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Modal from "components/Modal";
import { notify } from "components/ToastNotification";
import { IFormErrors } from "hooks/useFormValidation";
import {
  ICertificateAuthorityPartial,
  ICertificateAuthorityType,
} from "interfaces/certificates";
import certificatesAPI from "services/entities/certificates";

import CustomESTForm, {
  ICustomESTFormData,
} from "../CustomESTForm/CustomESTForm";
import CustomSCEPForm from "../CustomSCEPForm";
import { ICustomSCEPFormData } from "../CustomSCEPForm/CustomSCEPForm";
import DigicertForm from "../DigicertForm";
import { IDigicertFormData } from "../DigicertForm/DigicertForm";
import EJBCAForm from "../EJBCAForm";
import { IEJBCAFormData } from "../EJBCAForm/EJBCAForm";
import { handleEJBCAServerError } from "../EJBCAForm/helpers";
import GoogleCASForm from "../GoogleCASForm";
import { IGoogleCASFormData } from "../GoogleCASForm/GoogleCASForm";
import HydrantForm from "../HydrantForm";
import { IHydrantFormData } from "../HydrantForm/HydrantForm";
import NDESForm from "../NDESForm";
import { INDESFormData } from "../NDESForm/NDESForm";
import SmallstepForm, {
  ISmallstepFormData,
} from "../SmallstepForm/SmallstepForm";

import {
  generateAddCertAuthorityData,
  generateDropdownOptions,
  getErrorMessage,
} from "./helpers";

export type ICertFormData =
  | IDigicertFormData
  | IHydrantFormData
  | INDESFormData
  | ICustomSCEPFormData
  | ISmallstepFormData
  | ICustomESTFormData
  | IEJBCAFormData
  | IGoogleCASFormData;

const baseClass = "add-cert-authority-modal";

interface IAddCertAuthorityModalProps {
  certAuthorities: ICertificateAuthorityPartial[];
  onExit: () => void;
}

const AddCertAuthorityModal = ({
  certAuthorities,
  onExit,
}: IAddCertAuthorityModalProps) => {
  const dropdownOptions = useMemo(() => {
    return generateDropdownOptions(
      certAuthorities.some((cert) => cert.type === "ndes_scep_proxy")
    );
  }, [certAuthorities]);

  const [
    certAuthorityType,
    setCertAuthorityType,
  ] = useState<ICertificateAuthorityType>(
    dropdownOptions[0].value as ICertificateAuthorityType
  );
  const [isAdding, setIsAdding] = useState(false);
  const [
    ejbcaServerErrors,
    setEJBCAServerErrors,
  ] = useState<IFormErrors | null>(null);
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

  const [
    customESTFormData,
    setCustomESTFormData,
  ] = useState<ICustomESTFormData>({
    name: "",
    url: "",
    username: "",
    password: "",
  });

  const onChangeDropdown = (value: ICertificateAuthorityType) => {
    setCertAuthorityType(value);
    // the EJBCA form starts empty when it is shown again, so the errors of the last request don't belong to it
    setEJBCAServerErrors(null);
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
      case "custom_est_proxy":
        setFormData = setCustomESTFormData;
        formData = customESTFormData;
        break;
      default:
        return;
    }

    (setFormData as React.Dispatch<React.SetStateAction<ICertFormData>>)({
      ...formData,
      [update.name]: update.value,
    });
  };

  const onAddCertAuthority = async (submittedFormData?: ICertFormData) => {
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
      case "custom_est_proxy":
        formData = customESTFormData;
        break;
      case "google_cas":
        // This form keeps its own data and passes it in on submit.
        if (!submittedFormData) {
          return;
        }
        formData = submittedFormData;
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
      notify.success("Successfully added your certificate authority.");
      onExit();
    } catch (e) {
      notify.error(getErrorMessage(e, certAuthorityType), { response: e });
    }
    setIsAdding(false);
  };

  const onAddEJBCA = async (formData: IEJBCAFormData) => {
    const addCertAuthorityData = generateAddCertAuthorityData(
      "ejbca",
      formData
    );
    if (!addCertAuthorityData) {
      return;
    }
    setIsAdding(true);
    try {
      await certificatesAPI.addCertificateAuthority(addCertAuthorityData);
      notify.success("Successfully added your certificate authority.");
      onExit();
    } catch (e) {
      handleEJBCAServerError(e, "add", setEJBCAServerErrors);
    }
    setIsAdding(false);
  };

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
      case "custom_est_proxy":
        return (
          <CustomESTForm
            formData={customESTFormData}
            certAuthorities={certAuthorities}
            submitBtnText={submitBtnText}
            isSubmitting={isAdding}
            onChange={onChangeForm}
            onSubmit={onAddCertAuthority}
            onCancel={onExit}
          />
        );
      case "ejbca":
        return (
          <EJBCAForm
            certAuthorities={certAuthorities}
            submitBtnText={submitBtnText}
            isSubmitting={isAdding}
            serverErrors={ejbcaServerErrors}
            onSubmit={onAddEJBCA}
            onCancel={onExit}
          />
        );
      case "google_cas":
        return (
          <GoogleCASForm
            certAuthorities={certAuthorities}
            submitBtnText={submitBtnText}
            isSubmitting={isAdding}
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
      <Dropdown
        options={dropdownOptions}
        value={certAuthorityType}
        className={`${baseClass}__cert-authority-dropdown`}
        onChange={onChangeDropdown}
        searchable={false}
      />
      {renderForm()}
    </Modal>
  );
};

export default AddCertAuthorityModal;
