import React, { useContext, useState } from "react";

import mdmAPI, { IEulaMetadataResponse } from "services/entities/mdm";
import { NotificationContext } from "context/notification";

import SettingsSection from "pages/admin/components/SettingsSection";

import EulaUploader from "./components/EulaUploader/EulaUploader";
import UploadedEulaView from "./components/UploadedEulaView/UploadedEulaView";
import DeleteEulaModal from "./components/DeleteEulaModal/DeleteEulaModal";

const baseClass = "eula-section";

interface IEulaSectionProps {
  eulaMetadata?: IEulaMetadataResponse;
  isEulaUploaded: boolean;
  onUpload: () => void;
  onDelete: () => void;
}

const EulaSection = ({
  eulaMetadata,
  isEulaUploaded,
  onUpload,
  onDelete,
}: IEulaSectionProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [showDeleteEulaModal, setShowDeleteEulaModal] = useState(false);

  const onDeleteEula = async () => {
    if (!eulaMetadata) return;

    try {
      await mdmAPI.deleteEULA(eulaMetadata.token);
      renderFlash("success", "Successfully deleted!");
    } catch {
      renderFlash("error", "Couldn’t delete. Please try again.");
    } finally {
      setShowDeleteEulaModal(false);
      onDelete();
    }
  };

  return (
    <SettingsSection
      className={baseClass}
      title="End user license agreement (EULA)"
    >
      <div className={`${baseClass}__content`}>
        {!isEulaUploaded || !eulaMetadata ? (
          <EulaUploader onUpload={onUpload} />
        ) : (
          <UploadedEulaView
            eulaMetadata={eulaMetadata}
            onDelete={() => setShowDeleteEulaModal(true)}
          />
        )}
      </div>
      {showDeleteEulaModal && (
        <DeleteEulaModal
          onDelete={onDeleteEula}
          onCancel={() => setShowDeleteEulaModal(false)}
        />
      )}
    </SettingsSection>
  );
};

export default EulaSection;
