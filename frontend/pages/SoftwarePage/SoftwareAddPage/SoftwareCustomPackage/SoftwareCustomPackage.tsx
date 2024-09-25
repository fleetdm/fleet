import React, { useContext } from "react";
import { InjectedRouter } from "react-router";
import { Location } from "history";

import PATHS from "router/paths";
import { buildQueryStringFromParams, QueryParams } from "utilities/url";
import softwareAPI from "services/entities/software";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";

import CustomLink from "components/CustomLink";

import PackageForm from "pages/SoftwarePage/components/PackageForm";
import { IPackageFormData } from "pages/SoftwarePage/components/PackageForm/PackageForm";

import { ISoftwareAddPageQueryParams } from "../SoftwareAddPage";
import AddSoftwareCustomPackageForm from "./AddSoftwareCustomPackageForm";
import { ICustomPackageAppFormData } from "./AddSoftwareCustomPackageForm/AddSoftwareCustomPackageForm";
import { getErrorMessage } from "./helpers";

const baseClass = "software-custom-package";

// 8 minutes + 15 seconds to account for extra roundtrip time.
export const UPLOAD_TIMEOUT = (8 * 60 + 15) * 1000;
export const MAX_FILE_SIZE_MB = 500;
export const MAX_FILE_SIZE_BYTES = MAX_FILE_SIZE_MB * 1024 * 1024;

interface ISoftwarePackageProps {
  currentTeamId: number;
  router: InjectedRouter;
  location: Location<ISoftwareAddPageQueryParams>;
  isSidePanelOpen: boolean;
  setSidePanelOpen: (isOpen: boolean) => void;
}

const SoftwareCustomPackage = ({
  currentTeamId,
  router,
  location,
  isSidePanelOpen,
  setSidePanelOpen,
}: ISoftwarePackageProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isUploading, setIsUploading] = React.useState(false);

  const onCancel = () => {
    router.push(
      `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams({
        team_id: currentTeamId,
      })}`
    );
  };

  const onSubmit = async (formData: IPackageFormData) => {
    console.log("submit", formData);
    setIsUploading(true);

    if (formData.software && formData.software.size > MAX_FILE_SIZE_BYTES) {
      renderFlash(
        "error",
        `Couldn't add. The maximum file size is ${MAX_FILE_SIZE_MB} MB.`
      );
      setIsUploading(false);
      return;
    }

    // Note: This TODO is copied to onSaveSoftwareChanges in EditSoftwareModal
    // TODO: confirm we are deleting the second sentence (not modifying it) for non-self-service installers
    try {
      await softwareAPI.addSoftwarePackage(
        formData,
        currentTeamId,
        UPLOAD_TIMEOUT
      );
      renderFlash(
        "success",
        <>
          <b>{formData.software?.name}</b> successfully added.
          {formData.selfService
            ? " The end user can install from Fleet Desktop."
            : ""}
        </>
      );

      const newQueryParams: QueryParams = { team_id: currentTeamId };
      if (formData.selfService) {
        newQueryParams.self_service = true;
      } else {
        newQueryParams.available_for_install = true;
      }
      router.push(
        `${PATHS.SOFTWARE_TITLES}?${buildQueryStringFromParams(newQueryParams)}`
      );
    } catch (e) {
      const reason = getErrorReason(e);
      if (
        reason.includes("Couldn't add. Fleet couldn't read the version from")
      ) {
        renderFlash(
          "error",
          <>
            {reason}{" "}
            <CustomLink
              newTab
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/read-package-version`}
              text="Learn more"
              iconColor="core-fleet-white"
            />
          </>
        );
      } else {
        renderFlash("error", getErrorMessage(e));
      }
    }
    setIsUploading(false);
  };

  return (
    <div className={baseClass}>
      <PackageForm
        showSchemaButton={!isSidePanelOpen}
        onClickShowSchema={() => setSidePanelOpen(true)}
        className={`${baseClass}__package-form`}
        isUploading={false}
        onCancel={onCancel}
        onSubmit={onSubmit}
      />
      {/* <AddSoftwareCustomPackageForm
        showSchemaButton={!isSidePanelOpen}
        onClickShowSchema={() => setSidePanelOpen(true)}
        onCancel={onCancel}
        onSubmit={onSubmit}
      /> */}
    </div>
  );
};

export default SoftwareCustomPackage;
