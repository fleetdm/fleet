import React, { useContext, useState, useEffect } from "react";
import { useQuery } from "react-query";
import classnames from "classnames";

import { ILabelSummary } from "interfaces/label";
import { ISoftwarePackage } from "interfaces/software";

import { NotificationContext } from "context/notification";
import softwareAPI, {
  MAX_FILE_SIZE_BYTES,
  MAX_FILE_SIZE_MB,
} from "services/entities/software";
import labelsAPI, { getCustomLabels } from "services/entities/labels";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import deepDifference from "utilities/deep_difference";
import { getFileDetails } from "utilities/file/fileUtils";

import FileProgressModal from "components/FileProgressModal";
import Modal from "components/Modal";

import PackageForm from "pages/SoftwarePage/components/PackageForm";
import { IPackageFormData } from "pages/SoftwarePage/components/PackageForm/PackageForm";
import {
  generateSelectedLabels,
  getCustomTarget,
  getTargetType,
} from "pages/SoftwarePage/components/PackageForm/helpers";

import { getErrorMessage } from "./helpers";
import ConfirmSaveChangesModal from "../ConfirmSaveChangesModal";

const baseClass = "edit-software-modal";

// Install type used on add but not edit
export type IEditPackageFormData = Omit<IPackageFormData, "installType">;

interface IEditSoftwareModalProps {
  softwareId: number;
  teamId: number;
  software: ISoftwarePackage; // TODO
  refetchSoftwareTitle: () => void;
  onExit: () => void;
}

const EditSoftwareModal = ({
  softwareId,
  teamId,
  software,
  onExit,
  refetchSoftwareTitle,
}: IEditSoftwareModalProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [editSoftwareModalClasses, setEditSoftwareModalClasses] = useState(
    baseClass
  );
  const [isUpdatingSoftware, setIsUpdatingSoftware] = useState(false);
  const [
    showConfirmSaveChangesModal,
    setShowConfirmSaveChangesModal,
  ] = useState(false);
  const [pendingUpdates, setPendingUpdates] = useState<IEditPackageFormData>({
    software: null,
    installScript: "",
    selfService: false,
    targetType: "",
    customTarget: "",
    labelTargets: {},
  });
  const [uploadProgress, setUploadProgress] = useState(0);

  const {
    data: labels,
    isLoading: isLoadingLabels,
    isError: isErrorLabels,
  } = useQuery<ILabelSummary[], Error>(
    ["custom_labels"],
    () => labelsAPI.summary().then((res) => getCustomLabels(res.labels)),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );

  // Work around to not lose Edit Software modal data when Save changes modal opens
  // by using CSS to hide Edit Software modal when Save changes modal is open
  useEffect(() => {
    setEditSoftwareModalClasses(
      classnames(baseClass, {
        [`${baseClass}--hidden`]:
          showConfirmSaveChangesModal ||
          (!!pendingUpdates.software && isUpdatingSoftware),
      })
    );
  }, [
    showConfirmSaveChangesModal,
    pendingUpdates.software,
    isUpdatingSoftware,
  ]);

  useEffect(() => {
    const beforeUnloadHandler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      // Next line with e.returnValue is included for legacy support
      // e.g.Chrome / Edge < 119
      e.returnValue = true;
    };

    // set up event listener to prevent user from leaving page while uploading
    if (isUpdatingSoftware) {
      addEventListener("beforeunload", beforeUnloadHandler);
    } else {
      removeEventListener("beforeunload", beforeUnloadHandler);
    }

    // clean up event listener and timeout on component unmount
    return () => {
      removeEventListener("beforeunload", beforeUnloadHandler);
    };
  }, [isUpdatingSoftware]);

  const toggleConfirmSaveChangesModal = () => {
    // open and closes save changes modal
    setShowConfirmSaveChangesModal(!showConfirmSaveChangesModal);
  };

  const onSaveSoftwareChanges = async (formData: IEditPackageFormData) => {
    setIsUpdatingSoftware(true);

    if (formData.software && formData.software.size > MAX_FILE_SIZE_BYTES) {
      renderFlash(
        "error",
        `Couldn't edit software. The maximum file size is ${MAX_FILE_SIZE_MB} MB.`
      );
      setIsUpdatingSoftware(false);
      return;
    }

    // Note: This TODO is copied over from onAddPackage on AddPackage.tsx
    // TODO: confirm we are deleting the second sentence (not modifying it) for non-self-service installers
    try {
      await softwareAPI.editSoftwarePackage({
        data: formData,
        orignalPackage: software,
        softwareId,
        teamId,
        onUploadProgress: (progressEvent) => {
          const progress = progressEvent.progress || 0;
          // for large uploads it seems to take a bit for the server to finalize its response so we'll keep the
          // progress bar at 97% until the server response is received
          setUploadProgress(Math.max(progress - 0.03, 0.01));
        },
      });

      renderFlash(
        "success",
        <>
          Successfully edited <b>{formData.software?.name}</b>.
          {formData.selfService
            ? " The end user can install from Fleet Desktop."
            : ""}
        </>
      );
      onExit();
      refetchSoftwareTitle();
    } catch (e) {
      renderFlash("error", getErrorMessage(e, software));
    }
    setIsUpdatingSoftware(false);
  };

  const onEditSoftware = (formData: IPackageFormData) => {
    // Check for changes to conditionally confirm save changes modal
    const updates = deepDifference(formData, {
      software,
      installScript: software.install_script || "",
      preInstallQuery: software.pre_install_query || "",
      postInstallScript: software.post_install_script || "",
      uninstallScript: software.uninstall_script || "",
      selfService: software.self_service || false,
      targetType: getTargetType(software),
      customTarget: getCustomTarget(software),
      labelTargets: generateSelectedLabels(software),
    });

    setPendingUpdates(formData);

    const onlySelfServiceUpdated =
      Object.keys(updates).length === 1 && "selfService" in updates;
    if (onlySelfServiceUpdated) {
      // Proceed with saving changes (API expects only changes)
      onSaveSoftwareChanges(formData);
    } else {
      // Open the confirm save changes modal
      setShowConfirmSaveChangesModal(true);
    }
  };

  const onConfirmSoftwareChanges = () => {
    setShowConfirmSaveChangesModal(false);
    onSaveSoftwareChanges(pendingUpdates);
  };

  return (
    <>
      <Modal
        className={editSoftwareModalClasses}
        title="Edit software"
        onExit={onExit}
        width="large"
      >
        <PackageForm
          labels={labels ?? []}
          className={`${baseClass}__package-form`}
          isEditingSoftware
          onCancel={onExit}
          onSubmit={onEditSoftware}
          defaultSoftware={software}
          defaultInstallScript={software.install_script}
          defaultPreInstallQuery={software.pre_install_query}
          defaultPostInstallScript={software.post_install_script}
          defaultUninstallScript={software.uninstall_script}
          defaultSelfService={software.self_service}
        />
      </Modal>
      {showConfirmSaveChangesModal && (
        <ConfirmSaveChangesModal
          onClose={toggleConfirmSaveChangesModal}
          softwarePackageName={software?.name}
          onSaveChanges={onConfirmSoftwareChanges}
        />
      )}
      {!!pendingUpdates.software && isUpdatingSoftware && (
        <FileProgressModal
          fileDetails={getFileDetails(pendingUpdates.software)}
          fileProgress={uploadProgress}
        />
      )}
    </>
  );
};

export default EditSoftwareModal;
