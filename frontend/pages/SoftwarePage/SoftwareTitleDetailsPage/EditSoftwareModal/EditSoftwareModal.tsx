import React, { useContext, useState, useEffect } from "react";
import { InjectedRouter } from "react-router";
import classnames from "classnames";
import deepDifference from "utilities/deep_difference";

import { getErrorReason } from "interfaces/errors";

import PATHS from "router/paths";
import { NotificationContext } from "context/notification";
import softwareAPI from "services/entities/software";
import { QueryParams, buildQueryStringFromParams } from "utilities/url";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import CustomLink from "components/CustomLink";
import Modal from "components/Modal";

import PackageForm from "pages/SoftwarePage/components/PackageForm";
import { IPackageFormData } from "pages/SoftwarePage/components/PackageForm/PackageForm";
import {
  UPLOAD_TIMEOUT,
  MAX_FILE_SIZE_MB,
  MAX_FILE_SIZE_BYTES,
} from "pages/SoftwarePage/components/AddPackage/AddPackage";
import { getErrorMessage } from "./helpers";
import ConfirmSaveChangesModal from "../ConfirmSaveChangesModal";

const baseClass = "edit-software-modal";

interface IEditSoftwareModalProps {
  softwareId: number;
  teamId: number;
  router: InjectedRouter;
  software?: any; // TODO

  onExit: () => void;
  setAddedSoftwareToken: (token: string) => void;
}

const EditSoftwareModal = ({
  softwareId,
  teamId,
  router,
  software,
  onExit,
  setAddedSoftwareToken,
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
  const [pendingUpdates, setPendingUpdates] = useState<IPackageFormData>({
    software: null,
    installScript: "",
    selfService: false,
  });

  // Work around to not lose Edit Software modal data when Save changes modal opens
  // by using CSS to hide Edit Software modal when Save changes modal is open
  useEffect(() => {
    setEditSoftwareModalClasses(
      classnames(baseClass, {
        [`${baseClass}--hidden`]: showConfirmSaveChangesModal,
      })
    );
  }, [showConfirmSaveChangesModal]);

  useEffect(() => {
    let timeout: NodeJS.Timeout;

    const beforeUnloadHandler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      // Next line with e.returnValue is included for legacy support
      // e.g.Chrome / Edge < 119
      e.returnValue = true;
    };

    // set up event listener to prevent user from leaving page while uploading
    if (isUpdatingSoftware) {
      addEventListener("beforeunload", beforeUnloadHandler);
      timeout = setTimeout(() => {
        removeEventListener("beforeunload", beforeUnloadHandler);
      }, UPLOAD_TIMEOUT);
    } else {
      removeEventListener("beforeunload", beforeUnloadHandler);
    }

    // clean up event listener and timeout on component unmount
    return () => {
      removeEventListener("beforeunload", beforeUnloadHandler);
      clearTimeout(timeout);
    };
  }, [isUpdatingSoftware]);

  const toggleConfirmSaveChangesModal = () => {
    // open and closes save changes modal
    setShowConfirmSaveChangesModal(!showConfirmSaveChangesModal);
  };

  const onSaveSoftwareChanges = async (formData: IPackageFormData) => {
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
      await softwareAPI.editSoftwarePackage(
        formData,
        softwareId,
        teamId,
        UPLOAD_TIMEOUT
      );

      renderFlash(
        "success",
        <>
          Successfully edited <b>{formData.software?.name}</b>.
          {formData.selfService
            ? " The end user can install from Fleet Desktop."
            : ""}
        </>
      );
      const newQueryParams: QueryParams = { team_id: teamId };
      if (formData.selfService) {
        newQueryParams.self_service = true;
      } else {
        newQueryParams.available_for_install = true;
      }
      // any unique string - triggers SW refetch
      setAddedSoftwareToken(`${Date.now()}`);
      onExit();
      router.push(
        `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams(newQueryParams)}`
      );
    } catch (e) {
      const reason = getErrorReason(e);
      if (reason.includes("Fleet couldn't read the version from")) {
        renderFlash(
          "error",
          <>
            Couldn&apos;t edit <b>{software.name}</b>. {reason}.
            <CustomLink
              newTab
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/read-package-version`}
              text="Learn more"
            />
          </>
        );
      } else if (reason.includes("selected package is")) {
        renderFlash(
          "error",
          <>
            Couldn&apos;t edit <b>{software.name}</b>. {reason}
          </>
        );
      } else {
        renderFlash("error", getErrorMessage(e));
      }
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
    });

    setPendingUpdates(formData);

    const onlySelfServiceUpdated =
      Object.keys(updates).length === 1 && "selfService" in updates;
    if (!onlySelfServiceUpdated) {
      console.log("non-self-service updates: ", updates);
      // Open the confirm save changes modal
      setShowConfirmSaveChangesModal(true);
    } else {
      // Proceed with saving changes (API expects only changes)
      onSaveSoftwareChanges(formData);
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
      >
        <PackageForm
          isEditingSoftware
          isUploading={isUpdatingSoftware}
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
    </>
  );
};

export default EditSoftwareModal;
