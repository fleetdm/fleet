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
  currentTeamId: number;
}

const BootstrapPackage = ({ currentTeamId }: IBootstrapPackageProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [showDeletePackageModal, setShowDeletePackageModal] = useState(false);

  const {
    data: bootstrapMetadata,
    isLoading,
    isError,
    status,
    refetch: refretchBootstrapMetaData,
  } = useQuery(
    ["bootstrap-metadata", currentTeamId],
    () => mdmAPI.getBootstrapPackageMetadata(currentTeamId),
    {
      retry: false,
      refetchOnWindowFocus: false,
      onError: (e) => {
        // setPageState("error");
      },
      onSuccess: (e) => {
        // setPageState("packageUploaded")
      },
    }
  );

  const onUpload = () => {
    refretchBootstrapMetaData();
  };

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
      {isLoading ? (
        <Spinner />
      ) : (
        <div className={`${baseClass}__content`}>
          {bootstrapMetadata ? (
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
      )}
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
