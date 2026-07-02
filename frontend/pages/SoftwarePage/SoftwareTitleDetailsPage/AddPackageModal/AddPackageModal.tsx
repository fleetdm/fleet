import React, { useState } from "react";
import { useQuery, useQueryClient } from "react-query";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getFileDetails, IFileDetails } from "utilities/file/fileUtils";
import softwareAPI from "services/entities/software";
import labelsAPI, { getCustomLabels } from "services/entities/labels";

import useBlockNavigation from "hooks/useBlockNavigation";
import useGitOpsMode from "hooks/useGitOpsMode";
import { ILabelSummary } from "interfaces/label";

import { notify } from "components/ToastNotification";
import Modal from "components/Modal";
import FileProgressModal from "components/FileProgressModal";
import CategoriesEndUserExperienceModal from "pages/SoftwarePage/components/modals/CategoriesEndUserExperienceModal";

import PackageForm from "pages/SoftwarePage/components/forms/PackageForm";
import { IPackageFormData } from "pages/SoftwarePage/components/forms/PackageForm/PackageForm";

import { getErrorMessage } from "pages/SoftwarePage/SoftwareAddPage/SoftwareCustomPackage/helpers";

import { getFileTypeRestriction } from "./helpers";

const baseClass = "add-package-modal";

interface IAddPackageModalProps {
  /** The id of the software title we're adding a package to (multi-package
   * flow, #48397). The POST carries this as `software_title_id` so the new
   * package attaches to an existing title instead of creating a new one. */
  softwareTitleId: number;
  teamId: number;
  /** File name of the title's first-added package — used to derive the
   * platform/file-type restriction so the new upload matches the existing
   * package's platform (e.g. ".pkg" only when the title is a macOS title). */
  existingPackageName: string;
  onExit: () => void;
  /** Fires after a successful upload so the caller can refetch the title's
   * `packages[]` and surface the new row. */
  onSuccess: () => void;
}

const AddPackageModal = ({
  softwareTitleId,
  teamId,
  existingPackageName,
  onExit,
  onSuccess,
}: IAddPackageModalProps) => {
  const queryClient = useQueryClient();
  const { gitOpsModeEnabled } = useGitOpsMode("software");
  const restriction = getFileTypeRestriction(existingPackageName);

  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploadDetails, setUploadDetails] = useState<IFileDetails | null>(null);
  const [
    showPreviewEndUserExperience,
    setShowPreviewEndUserExperience,
  ] = useState(false);
  const [
    isIpadOrIphoneSoftwareSource,
    setIsIpadOrIphoneSoftwareSource,
  ] = useState(false);

  const { data: labels } = useQuery<ILabelSummary[], Error>(
    ["custom_labels"],
    () => labelsAPI.summary(teamId).then((res) => getCustomLabels(res.labels)),
    { ...DEFAULT_USE_QUERY_OPTIONS }
  );

  // Block tab close / hard navigation while an upload is in flight so the
  // user doesn't lose their work mid-request.
  useBlockNavigation(!!uploadDetails);

  const onClickPreviewEndUserExperience = (isIosOrIpadosApp = false) => {
    setShowPreviewEndUserExperience(!showPreviewEndUserExperience);
    setIsIpadOrIphoneSoftwareSource(isIosOrIpadosApp);
  };

  const onSubmit = async (formData: IPackageFormData) => {
    if (!formData.software) {
      notify.error("Couldn't add. Please refresh the page and try again.");
      return;
    }

    setUploadDetails(getFileDetails(formData.software));

    try {
      await softwareAPI.addSoftwarePackage({
        data: formData,
        teamId,
        softwareTitleId,
        onUploadProgress: (progressEvent) => {
          const progress = progressEvent.progress || 0;
          // Keep the progress bar at 97% until the server finalizes its
          // response — large uploads stall on the last few percent otherwise.
          setUploadProgress(Math.max(progress - 0.03, 0.01));
        },
      });

      if (!gitOpsModeEnabled) {
        notify.success(
          <>
            Successfully added new <b>{formData.software.name}</b> package.
          </>
        );
      }

      queryClient.invalidateQueries({
        queryKey: [{ scope: "software-titles" }],
      });
      queryClient.invalidateQueries({
        queryKey: [{ scope: "software-library" }],
      });

      onSuccess();
    } catch (e) {
      notify.error(getErrorMessage(e), { response: e });
    }
    setUploadDetails(null);
  };

  return (
    <>
      <Modal
        className={baseClass}
        title="Add package"
        onExit={onExit}
        width="large"
      >
        <PackageForm
          labels={labels || []}
          className={`${baseClass}__package-form`}
          onCancel={onExit}
          onSubmit={onSubmit}
          onClickPreviewEndUserExperience={onClickPreviewEndUserExperience}
          multiPackageContext
          restrictedFileAccept={restriction?.accept}
          restrictedFileTypeLabel={restriction?.label}
          initialTargetType="Custom"
        />
      </Modal>
      {uploadDetails && (
        <FileProgressModal
          fileDetails={uploadDetails}
          fileProgress={uploadProgress}
        />
      )}
      {showPreviewEndUserExperience && (
        <CategoriesEndUserExperienceModal
          onCancel={onClickPreviewEndUserExperience}
          teamId={teamId}
          isIosOrIpadosApp={isIpadOrIphoneSoftwareSource}
        />
      )}
    </>
  );
};

export default AddPackageModal;
