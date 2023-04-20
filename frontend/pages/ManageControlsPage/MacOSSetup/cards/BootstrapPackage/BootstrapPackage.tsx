import React, { useContext, useState } from "react";
import { useQuery } from "react-query";

import mdmAPI from "services/entities/mdm";
import { NotificationContext } from "context/notification";

import Spinner from "components/Spinner";
import BootstrapPackagePreview from "./components/BootstrapPackagePreview";
import PackageUploader from "./components/BootstrapPackageUploader";
import UploadedPackageView from "./components/UploadedPackageView";
import DeletePackageModal from "./components/DeletePackageModal/DeletePackageModal";

const baseClass = "bootstrap-package";

interface IBootstrapPackageProps {
  bootstrapConfigured: boolean;
  isLoading: boolean;
  currentTeamId: number;
}

const BootstrapPackage = ({
  bootstrapConfigured,
  isLoading,
  currentTeamId,
}: IBootstrapPackageProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [showDeletePackageModal, setShowDeletePackageModal] = useState(false);

  const onUpload = () => {};

  const onDelete = async () => {
    try {
      await mdmAPI.deleteBootstrapPackage(currentTeamId);
      renderFlash("success", "Successfully deleted!");
    } catch {
      renderFlash("error", "Couldn’t delete. Please try again.");
    } finally {
      setShowDeletePackageModal(false);
    }
  };

  return (
    <div className={baseClass}>
      <h2>Bootstrap package</h2>
      <div className={`${baseClass}__content`}>
        {isLoading && <Spinner />}
        {bootstrapConfigured ? (
          <>
            <UploadedPackageView
              currentTeamId={currentTeamId}
              onDelete={() => setShowDeletePackageModal(true)}
            />
            <div className={`${baseClass}__preview-container`}>
              <BootstrapPackagePreview />
            </div>
          </>
        ) : (
          <>
            <PackageUploader
              currentTeamId={currentTeamId}
              onUpload={onUpload}
            />
            <div className={`${baseClass}__preview-container`}>
              <BootstrapPackagePreview />
            </div>
          </>
        )}
      </div>
      {showDeletePackageModal && (
        <DeletePackageModal
          onDelete={onDelete}
          onCancel={() => setShowDeletePackageModal(false)}
        />
      )}
    </div>
  );
};

export default BootstrapPackage;
