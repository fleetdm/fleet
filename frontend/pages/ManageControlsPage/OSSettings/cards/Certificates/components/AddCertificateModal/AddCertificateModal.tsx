import { noop } from "lodash";
import React, { useContext, useMemo, useState } from "react";
import { useQuery } from "react-query";
import { SingleValue } from "react-select-5";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import FileUploader from "components/FileUploader";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import InputField from "components/forms/fields/InputField";
import Radio from "components/forms/fields/Radio";
import Modal from "components/Modal";
import Spinner from "components/Spinner";
import {
  TargetLabelSelector,
  LabelTargetMode,
  TargetType,
} from "components/TargetLabelSelector";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import { getErrorReason } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";
import paths from "router/paths";
import certificatesAPI from "services/entities/certificates";
import labelsAPI, { getCustomLabels } from "services/entities/labels";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  MAX_ENTITY_CHAR_LENGTH,
} from "utilities/constants";

import { generateCustomTargetLabelKey } from "../../../ConfigurationProfiles/components/ProfileUploader/helpers";

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
  subjectAlternativeName: string;
}

interface IAddCertModalProps {
  onExit: () => void;
  onSuccess: () => void;
  currentTeamId?: number;
}

type CertModalType = "scep" | "ca";

const AddCertModal = ({
  onExit,
  onSuccess,
  currentTeamId,
}: IAddCertModalProps) => {
  const { isPremiumTier } = useContext(AppContext);
  const [certType, setCertType] = useState<CertModalType>("scep");
  const [isUpdating, setIsUpdating] = useState(false);
  const [attemptedSubmit, setAttemptedSubmit] = useState(false);
  const [formData, setFormData] = useState<IAddCertFormData>({
    name: "",
    certAuthorityId: "",
    subjectName: "",
    subjectAlternativeName: "",
  });

  // CA certificate state
  const [caFile, setCaFile] = useState<File | null>(null);
  const [caName, setCaName] = useState("");
  const [selectedTargetType, setSelectedTargetType] = useState<TargetType>(
    "All hosts"
  );
  const [selectedIncludeLabels, setSelectedIncludeLabels] = useState<
    Record<string, boolean>
  >({});
  const [selectedExcludeLabels, setSelectedExcludeLabels] = useState<
    Record<string, boolean>
  >({});
  const [
    selectedLabelIncludeMode,
    setSelectedLabelIncludeMode,
  ] = useState<LabelTargetMode>("any");

  const onSelectIncludeLabel = ({
    name,
    value,
  }: {
    name: string;
    value: boolean;
  }) => {
    setSelectedIncludeLabels((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const onSelectExcludeLabel = ({
    name,
    value,
  }: {
    name: string;
    value: boolean;
  }) => {
    setSelectedExcludeLabels((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const { data: labels, isLoading: isLoadingLabels } = useQuery<
    ILabelSummary[],
    Error
  >(
    ["custom_labels", currentTeamId],
    () =>
      labelsAPI
        .summary(currentTeamId)
        .then((res) => getCustomLabels(res.labels)),
    {
      enabled: isPremiumTier && certType === "ca",
      refetchOnWindowFocus: false,
      retry: false,
      staleTime: 10000,
    }
  );

  // Server-side validation errors keyed by form field; cleared when the user
  // edits the corresponding input.
  const [serverErrors, setServerErrors] = useState<{
    name?: string;
    subjectAlternativeName?: string;
  }>({});

  const validations = useMemo(() => generateFormValidations(), []);

  // formValidation is derived from formData + attemptedSubmit; computing it during render via
  // useMemo keeps it in lockstep with its inputs without scattering setFormValidation calls
  // across handlers.
  // See https://react.dev/learn/choosing-the-state-structure#avoid-redundant-state.
  const formValidation: IAddCertFormValidation = useMemo(
    () => validateFormData(formData, validations, attemptedSubmit),
    [formData, validations, attemptedSubmit]
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

  // Only custom SCEP CAs are supported for Android certificate profiles.
  const caDropdownOptions = caPartials
    .filter((cAP) => cAP.type === "custom_scep_proxy")
    .map((cAP) => ({
      value: cAP.id.toString(),
      label: cAP.name,
    }));

  const onInputChange = (update: { name: string; value: string }) => {
    const updatedFormData = { ...formData, [update.name]: update.value };
    setFormData(updatedFormData);
    if (update.name === "name" && serverErrors.name) {
      setServerErrors((prev) => ({ ...prev, name: undefined }));
    }
    if (
      update.name === "subjectAlternativeName" &&
      serverErrors.subjectAlternativeName
    ) {
      setServerErrors((prev) => ({
        ...prev,
        subjectAlternativeName: undefined,
      }));
    }
  };

  const onChangeCA = (newValue: SingleValue<CustomOptionType>) => {
    const updatedFormData = {
      ...formData,
      certAuthorityId: newValue?.value ?? "",
    };
    setFormData(updatedFormData);
  };

  const onSubmitForm = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    if (!formValidation.isValid) {
      setAttemptedSubmit(true);
      return;
    }

    setIsUpdating(true);
    try {
      await certificatesAPI.addCert({
        name: formData.name,
        certAuthorityId: parseInt(formData.certAuthorityId, 10),
        subjectName: formData.subjectName,
        subjectAlternativeName: formData.subjectAlternativeName,
        teamId: currentTeamId,
      });
      notify.success("Successfully added your certificate.");
      onSuccess();
      onExit();
    } catch (e) {
      const sanReason = getErrorReason(e, {
        nameEquals: "subject_alternative_name",
      });
      const nameConflict = getErrorReason(e, {
        reasonIncludes: "already exists",
      });
      if (sanReason) {
        setServerErrors({ subjectAlternativeName: sanReason });
      } else if (nameConflict) {
        setServerErrors({
          name: "Name is already used by another certificate.",
        });
      } else {
        notify.error("Couldn't add certificate. Please try again.", {
          response: e,
        });
      }
    } finally {
      setIsUpdating(false);
    }
  };

  const onSubmitCAForm = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    if (!caFile) {
      notify.error("Please choose a certificate file to upload.");
      return;
    }

    setIsUpdating(true);
    try {
      const labelKey = generateCustomTargetLabelKey({
        targetType: selectedTargetType,
        includeMode: selectedLabelIncludeMode,
        includeLabels: selectedIncludeLabels,
        excludeLabels: selectedExcludeLabels,
      });

      await certificatesAPI.uploadCACert({
        file: caFile,
        name: caName.trim() || undefined,
        teamId: currentTeamId,
        labelsIncludeAll: labelKey.labelsIncludeAll,
        labelsIncludeAny: labelKey.labelsIncludeAny,
        labelsExcludeAny: labelKey.labelsExcludeAny,
      });

      notify.success("Successfully added your CA certificate.");
      onSuccess();
      onExit();
    } catch (e) {
      notify.error("Couldn't add certificate. Please try again.", {
        response: e,
      });
    } finally {
      setIsUpdating(false);
    }
  };

  const renderCAForm = () => {
    return (
      <form className={baseClass} onSubmit={onSubmitCAForm}>
        <FileUploader
          graphicName="file-pem"
          accept=".pem,.crt"
          message="Upload a .pem or .crt certificate file."
          buttonMessage="Choose file"
          onFileUpload={(files) => setCaFile(files ? files[0] : null)}
          onDeleteFile={() => setCaFile(null)}
          fileDetails={caFile ? { name: caFile.name } : undefined}
        />
        <InputField
          name="caName"
          label="Name (optional)"
          value={caName}
          onChange={({ value }) => setCaName(value)}
          placeholder="Corporate Root CA"
          helpText="Optional. If omitted, the certificate's common name will be used."
          parseTarget
          inputOptions={{ maxLength: MAX_ENTITY_CHAR_LENGTH }}
        />
        <TargetLabelSelector
          labels={labels ?? []}
          isLoadingLabels={isLoadingLabels}
          emptyStateDescription="Add a label to target your certificate."
          onAddLabel={noop}
          selectedTargetType={selectedTargetType}
          onSelectTargetType={setSelectedTargetType}
          includeConfig={{
            selectedLabels: selectedIncludeLabels,
            onSelectLabel: onSelectIncludeLabel,
            showModeToggle: true,
            mode: selectedLabelIncludeMode,
            onSelectMode: setSelectedLabelIncludeMode,
          }}
          excludeConfig={{
            selectedLabels: selectedExcludeLabels,
            onSelectLabel: onSelectExcludeLabel,
          }}
        />
        <div className="modal-cta-wrap">
          <Button
            isLoading={isUpdating}
            disabled={isUpdating || !caFile}
            type="submit"
          >
            Upload
          </Button>
          <Button variant="secondary" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </form>
    );
  };

  const renderSCEPForm = () => {
    if (isLoadingCAs) {
      return <Spinner />;
    }

    if (isErrorCAs) {
      return <DataError />;
    }
    return (
      <form className={baseClass} onSubmit={onSubmitForm}>
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
          name="name"
          label="Name"
          value={formData.name}
          onChange={onInputChange}
          error={serverErrors.name ?? formValidation.name?.message}
          helpText="Letters, numbers, spaces, dashes, and underscores only. Name can be used as certificate alias to reference in configuration profiles."
          parseTarget
          placeholder="VPN certificate"
          inputOptions={{ maxLength: MAX_ENTITY_CHAR_LENGTH }}
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
        <InputField
          name="subjectAlternativeName"
          label="Subject alternative name (SAN)"
          type="textarea"
          value={formData.subjectAlternativeName}
          onChange={onInputChange}
          error={
            serverErrors.subjectAlternativeName ??
            formValidation.subjectAlternativeName?.message
          }
          helpText='Optional. Separate fields by ", " using format KEY=value. Allowed keys: DNS, EMAIL, UPN, IP, URI.'
          parseTarget
          placeholder="UPN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME, EMAIL=$FLEET_VAR_HOST_END_USER_IDP_USERNAME"
        />
        <div className="modal-cta-wrap">
          <Button isLoading={isUpdating} disabled={isUpdating} type="submit">
            Add
          </Button>
          <Button variant="secondary" onClick={onExit}>
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
      <div className={`${baseClass}__cert-type-radios`}>
        <Radio
          label="Client certificate (SCEP)"
          id="cert-type-scep"
          checked={certType === "scep"}
          value="scep"
          name="certType"
          onChange={() => setCertType("scep")}
        />
        <Radio
          label="Root or intermediate CA (.pem, .crt)"
          id="cert-type-ca"
          checked={certType === "ca"}
          value="ca"
          name="certType"
          onChange={() => setCertType("ca")}
        />
      </div>
      {certType === "scep" ? renderSCEPForm() : renderCAForm()}
    </Modal>
  );
};

export default AddCertModal;
