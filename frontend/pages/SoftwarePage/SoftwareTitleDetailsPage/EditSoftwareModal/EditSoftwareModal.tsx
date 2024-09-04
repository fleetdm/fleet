import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";

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

const baseClass = "edit-software-modal";

interface IEditSoftwareModalProps {
  softwareId: number;
  teamId: number;
  router: InjectedRouter;
  software?: any; // TODO
  installScript?: string;
  preInstallQuery?: string;
  postInstallScript?: string;
  uninstallScript?: string;
  selfService?: boolean;
  onExit: () => void;
  setAddedSoftwareToken: (token: string) => void;
}

const EditSoftwareModal = ({
  softwareId,
  teamId,
  router,
  software,
  installScript,
  preInstallQuery,
  postInstallScript,
  uninstallScript,
  selfService,
  onExit,
  setAddedSoftwareToken,
}: IEditSoftwareModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUpdatingSoftware, setIsUpdatingSoftware] = useState(false);

  const onEditSoftware = async (formData: IAddPackageFormData) => {
    setIsUpdatingSoftware(true);

    if (formData.software && formData.software.size > MAX_FILE_SIZE_BYTES) {
      renderFlash(
        "error",
        `Couldn't edit software. The maximum file size is ${MAX_FILE_SIZE_MB} MB.`
      );
      setIsUpdatingSoftware(false);
      return;
    }

    // TODO: confirm we are deleting the second sentence (not modifying it) for non-self-service installers
    try {
      await softwareAPI.editSoftwarePackage(
        softwareId,
        formData,
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

  return (
    <Modal className={baseClass} title="Edit software" onExit={onExit}>
      <AddPackageForm
        isEditingSoftware
        isUploading={isUpdatingSoftware}
        onCancel={onExit}
        onSubmit={onEditSoftware}
        defaultSoftware={software}
        defaultInstallScript={installScript}
        defaultPreInstallQuery={preInstallQuery}
        defaultPostInstallScript={postInstallScript}
        defaultUninstallScript={uninstallScript}
        defaultSelfService={selfService}
      />
    </Modal>
  );
};

export default EditSoftwareModal;
