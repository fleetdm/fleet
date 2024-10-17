import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { AxiosResponse } from "axios";

import { IBootstrapPackageMetadata } from "interfaces/mdm";
import { IApiError } from "interfaces/errors";
import mdmAPI from "services/entities/mdm";
import { NotificationContext } from "context/notification";

import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";

import BootstrapPackagePreview from "./components/BootstrapPackagePreview";
import PackageUploader from "./components/BootstrapPackageUploader";
import UploadedPackageView from "./components/UploadedPackageView";
import DeletePackageModal from "./components/DeletePackageModal";

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
    error,
    refetch: refretchBootstrapMetadata,
  } = useQuery<
    IBootstrapPackageMetadata,
    AxiosResponse<IApiError>,
    IBootstrapPackageMetadata
  >(
    ["bootstrap-metadata", currentTeamId],
    () => mdmAPI.getBootstrapPackageMetadata(currentTeamId),
    {
      retry: false,
      refetchOnWindowFocus: false,
      cacheTime: 0,
    }
  );

  const onUpload = () => {
    refretchBootstrapMetadata();
  };

  const onDelete = async () => {
    try {
      await mdmAPI.deleteBootstrapPackage(currentTeamId);
      renderFlash("success", "Successfully deleted!");
    } catch {
      renderFlash("error", "Couldn’t delete. Please try again.");
    } finally {
      setShowDeletePackageModal(false);
      refretchBootstrapMetadata();
    }
  };

  // we are relying on the API to tell us this resource does not exist to
  // determine if the user has uploaded a bootstrap package.
  const noPackageUploaded =
    (error && error.status === 404) || !bootstrapMetadata;

  return (
    <div className={baseClass}>
      <SectionHeader title="Bootstrap package" />
      {isLoading ? (
        <Spinner />
      ) : (
        <div className={`${baseClass}__content`}>
          {noPackageUploaded ? (
            <>
              <PackageUploader
                currentTeamId={currentTeamId}
                onUpload={onUpload}
              />
              <div className={`${baseClass}__preview-container`}>
                <BootstrapPackagePreview />
              </div>
            </>
          ) : (
            <>
              <UploadedPackageView
                bootstrapPackage={bootstrapMetadata}
                currentTeamId={currentTeamId}
                onDelete={() => setShowDeletePackageModal(true)}
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
