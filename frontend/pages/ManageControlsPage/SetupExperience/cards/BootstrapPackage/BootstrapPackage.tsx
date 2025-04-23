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
import DeleteBootstrapPackageModal from "./components/DeleteBootstrapPackageModal";
import SetupExperienceContentContainer from "../../components/SetupExperienceContentContainer";
import BootstrapAdvancedOptions from "./components/BootstrapAdvancedOptions";

const baseClass = "bootstrap-package";

interface IBootstrapPackageProps {
  currentTeamId: number;
}

const BootstrapPackage = ({ currentTeamId }: IBootstrapPackageProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [
    showDeleteBootstrapPackageModal,
    setShowDeleteBootstrapPackageModal,
  ] = useState(false);

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
      renderFlash("error", "Couldn't delete. Please try again.");
    } finally {
      setShowDeleteBootstrapPackageModal(false);
      refretchBootstrapMetadata();
    }
  };

  // we are relying on the API to tell us this resource does not exist to
  // determine if the user has uploaded a bootstrap package.
  const noPackageUploaded =
    (error && error.status === 404) || !bootstrapMetadata;

  const renderBootstrapView = () => {
    const bootstrapPackageView = noPackageUploaded ? (
      <PackageUploader currentTeamId={currentTeamId} onUpload={onUpload} />
    ) : (
      <UploadedPackageView
        bootstrapPackage={bootstrapMetadata}
        currentTeamId={currentTeamId}
        onDelete={() => setShowDeleteBootstrapPackageModal(true)}
      />
    );

    return (
      <SetupExperienceContentContainer className={`${baseClass}__content`}>
        <div className={`${baseClass}__uploader-container`}>
          {bootstrapPackageView}
          <BootstrapAdvancedOptions
            enableInstallManually={!noPackageUploaded}
            defaultInstallManually={false}
          />
        </div>
        <div className={`${baseClass}__preview-container`}>
          <BootstrapPackagePreview />
        </div>
      </SetupExperienceContentContainer>
    );
  };

  return (
    <section className={baseClass}>
      <SectionHeader title="Bootstrap package" />
      {isLoading ? <Spinner /> : renderBootstrapView()}
      {showDeleteBootstrapPackageModal && (
        <DeleteBootstrapPackageModal
          onDelete={onDelete}
          onCancel={() => setShowDeleteBootstrapPackageModal(false)}
        />
      )}
    </section>
  );
};

export default BootstrapPackage;
