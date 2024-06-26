import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import mdmAPI, { IEulaMetadataResponse } from "services/entities/mdm";
import { NotificationContext } from "context/notification";

import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";

import EulaUploader from "./components/EulaUploader/EulaUploader";
import UploadedEulaView from "./components/UploadedEulaView/UploadedEulaView";
import DeleteEulaModal from "./components/DeleteEulaModal/DeleteEulaModal";

const baseClass = "eula-section";

const EulaSection = () => {
  const { renderFlash } = useContext(NotificationContext);
  const [showDeleteEulaModal, setShowDeleteEulaModal] = useState(false);

  const {
    data: eulaMetadata,
    isLoading,
    error,
    refetch: refetchEulaMetadata,
  } = useQuery<IEulaMetadataResponse, AxiosResponse<IApiError>>(
    ["eula-metadata"],
    () => mdmAPI.getEULAMetadata(),
    {
      retry: false,
      refetchOnWindowFocus: false,
    }
  );

  const onUpload = () => {
    refetchEulaMetadata();
  };

  const onDelete = async () => {
    if (!eulaMetadata) return;

    try {
      await mdmAPI.deleteEULA(eulaMetadata.token);
      renderFlash("success", "Successfully deleted!");
    } catch {
      renderFlash("error", "Couldn’t delete. Please try again.");
    } finally {
      setShowDeleteEulaModal(false);
      refetchEulaMetadata();
    }
  };

  // we are relying on the API to tell us this resource does not exist to
  // determine if the user has uploaded a bootstrap package.
  const noEulaUploaded = (error && error.status === 404) || !eulaMetadata;

  return (
    <div className={baseClass}>
      <SectionHeader title="End user license agreement (EULA)" />
      {isLoading ? (
        <Spinner />
      ) : (
        <div className={`${baseClass}__content`}>
          {noEulaUploaded ? (
            <EulaUploader onUpload={onUpload} />
          ) : (
            <UploadedEulaView
              eulaMetadata={eulaMetadata}
              onDelete={() => setShowDeleteEulaModal(true)}
            />
          )}
        </div>
      )}
      {showDeleteEulaModal && (
        <DeleteEulaModal
          onDelete={onDelete}
          onCancel={() => setShowDeleteEulaModal(false)}
        />
      )}
    </div>
  );
};

export default EulaSection;
