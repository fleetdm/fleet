import React from "react";

// TODO: Rename this to PackageFormData after blocker PRs merge to avoid merge conflicts
import { IAddPackageFormData } from "pages/SoftwarePage/components/AddPackageForm/AddPackageForm";

// TODO: Rename AddPackageForm.tsx to PackageForm.tsx after blocker PRs merge to avoid merge conflicts
import AddPackageForm from "pages/SoftwarePage/components/AddPackageForm";
import Modal from "components/Modal";

const baseClass = "edit-software-modal";

interface IEditSoftwareModalProps {
  software?: any; // TODO
  installScript?: string;
  preInstallQuery?: string;
  postInstallScript?: string;
  uninstallScript?: string;
  selfService?: boolean;
  onEditSoftware: (formData: IAddPackageFormData) => void;
  isUpdatingSoftware: boolean;
  onExit: () => void;
}

const EditSoftwareModal = ({
  software,
  installScript,
  preInstallQuery,
  postInstallScript,
  uninstallScript,
  selfService,
  onEditSoftware,
  isUpdatingSoftware,
  onExit,
}: IEditSoftwareModalProps) => {
  return (
    <Modal className={baseClass} title="Edit software" onExit={onExit}>
      <AddPackageForm
        isEditingSoftware
        isUploading={isUpdatingSoftware}
        onCancel={onExit}
        onSubmit={onEditSoftware}
        defaultSoftware={software}
        defaultInstallScript={installScript}
        defaultPreInstallQuery={preInstallQuery}
        defaultPostInstallScript={postInstallScript}
        defaultUninstallScript={uninstallScript}
        defaultSelfService={selfService}
      />
    </Modal>
  );
};

export default EditSoftwareModal;
