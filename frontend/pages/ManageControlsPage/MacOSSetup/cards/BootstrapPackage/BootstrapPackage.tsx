import React, { useState } from "react";

import { IBootstrapPackage } from "interfaces/mdm";

import BootstrapPackagePreview from "./components/BootstrapPackagePreview";
import PackageUploader from "./components/PackageUploader";
import UploadedPackageView from "./components/UploadedPackageView";
import DeletePackageModal from "./components/DeletePackageModal/DeletePackageModal";

const baseClass = "bootstrap-package";

interface IBootstrapPackageProps {
  currentTeamId?: number;
}

const BootstrapPackage = ({ currentTeamId }: IBootstrapPackageProps) => {
  // TODO: get bootstrap package API call
  const [showDeletePackageModal, setShowDeletePackageModal] = useState(false);

  const onDelete = () => {};

  return (
    <div className={baseClass}>
      <h2>Bootstrap package</h2>
      <div className={`${baseClass}__content`}>
        {/* {bootstrapPackage ? <UploadedPackageView /> : <PackageUploader />} */}
        {true ? (
          <UploadedPackageView
            onDelete={() => setShowDeletePackageModal(true)}
          />
        ) : (
          <PackageUploader onUpload={() => {}} />
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
