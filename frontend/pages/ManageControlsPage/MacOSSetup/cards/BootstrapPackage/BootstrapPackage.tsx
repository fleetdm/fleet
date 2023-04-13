import React, { useState } from "react";

import { IBootstrapPackage } from "interfaces/mdm";

import BootstrapPackagePreview from "./components/BootstrapPackagePreview";
import PackageUploader from "./components/PackageUploader";
import UploadedPackageView from "./components/UploadedPackageView";

const baseClass = "bootstrap-package";

interface IBootstrapPackageProps {
  currentTeamId?: number;
}

const BootstrapPackage = ({ currentTeamId }: IBootstrapPackageProps) => {
  // TODO: get bootstrap package API call

  return (
    <div className={baseClass}>
      <h2>Bootstrap package</h2>
      <div className={`${baseClass}__content`}>
        {/* {bootstrapPackage ? <UploadedPackageView /> : <PackageUploader />} */}
        {true ? (
          <UploadedPackageView />
        ) : (
          <PackageUploader onUpload={() => {}} />
        )}
        <div className={`${baseClass}__preview-container`}>
          <BootstrapPackagePreview />
        </div>
      </div>
    </div>
  );
};

export default BootstrapPackage;
