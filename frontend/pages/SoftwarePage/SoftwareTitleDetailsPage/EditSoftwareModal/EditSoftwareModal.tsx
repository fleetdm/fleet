import React, { useContext, useState, useEffect } from "react";
import { InjectedRouter } from "react-router";
import classnames from "classnames";
import { noop } from "lodash";
import deepDifference from "utilities/deep_difference";

import { getErrorReason } from "interfaces/errors";

import PATHS from "router/paths";
import { NotificationContext } from "context/notification";
import softwareAPI from "services/entities/software";
import { QueryParams, buildQueryStringFromParams } from "utilities/url";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";

import CustomLink from "components/CustomLink";
import Modal from "components/Modal";

// TODO: Rename AddPackageForm.tsx to PackageForm.tsx after blocker PRs merge to avoid merge conflicts
import AddPackageForm from "pages/SoftwarePage/components/AddPackageForm";
// TODO: Rename this to PackageFormData after blocker PRs merge to avoid merge conflicts
import { IAddPackageFormData } from "pages/SoftwarePage/components/AddPackageForm/AddPackageForm";
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
  const [pendingUpdates, setPendingUpdates] = useState<IAddPackageFormData>({
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

  const toggleConfirmSaveChangesModal = () => {
    // open and closes save changes modal
    setShowConfirmSaveChangesModal(!showConfirmSaveChangesModal);
  };

  const onSaveSoftwareChanges = async (formData: IAddPackageFormData) => {
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
      console.log("Error: ", e);
      const reason = getErrorReason(e);
      if (
        reason.includes(
          "Couldn't edit software. Fleet couldn't read the version from"
        )
      ) {
        renderFlash(
          "error",
          `${reason}. ${(
            <CustomLink
              newTab
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/read-package-version`}
              text="Learn more"
            />
          )} `
        );
      }
      renderFlash("error", getErrorMessage(e));
    }
    setIsUpdatingSoftware(false);
  };

  const onEditSoftware = (formData: IAddPackageFormData) => {
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
    onSaveSoftwareChanges(pendingUpdates);
  };

  return (
    <>
      <Modal
        className={editSoftwareModalClasses}
        title="Edit software"
        onExit={onExit}
      >
        <AddPackageForm
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
