import React, { useContext, useState } from "react";
import { useQuery } from "react-query";

import mdmAPI from "services/entities/mdm";

import BootstrapPackagePreview from "./components/BootstrapPackagePreview";
import PackageUploader from "./components/BootstrapPackageUploader";
import UploadedPackageView from "./components/UploadedPackageView";
import DeletePackageModal from "./components/DeletePackageModal/DeletePackageModal";

const baseClass = "bootstrap-package";

interface IBootstrapPackageProps {
  bootstrapConfigured: boolean;
  currentTeamId: number;
}

const BootstrapPackage = ({
  bootstrapConfigured,
  currentTeamId,
}: IBootstrapPackageProps) => {
  // TODO: get bootstrap package API call
  const [hasBootstrapPackage, setHasBootstrapPackage] = useState(
    bootstrapConfigured
  );
  const [showDeletePackageModal, setShowDeletePackageModal] = useState(false);

  const {
    data: bootstrapMetadata,
    isLoading,
    isError,
    refetch: refretchBootstrapMetaData,
  } = useQuery(
    ["bootstrap-metadata", currentTeamId],
    () => {
      mdmAPI.getBootstrapPackageMetadata(currentTeamId);
    },
    {
      retry: false,
      refetchOnWindowFocus: false,
      enabled: hasBootstrapPackage,
    }
  );

  const onUpload = () => {
    refretchBootstrapMetaData();
  };

  const onDelete = () => {};

  return (
    <div className={baseClass}>
      <h2>Bootstrap package</h2>
      <div className={`${baseClass}__content`}>
        {!hasBootstrapPackage ? (
          <PackageUploader currentTeamId={currentTeamId} onUpload={onUpload} />
        ) : (
          <UploadedPackageView
            onDelete={() => setShowDeletePackageModal(true)}
          />
        )}
        <div className={`${baseClass}__preview-container`}>
          <BootstrapPackagePreview />
        </div>
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
