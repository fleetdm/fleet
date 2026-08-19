import React, { useRef, useState } from "react";
import { useQuery } from "react-query";
import { AxiosResponse } from "axios";

import PATHS from "router/paths";
import { notify } from "components/ToastNotification";

import { IApiError } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";
import { IMdmProfile, IProfileLabel } from "interfaces/mdm";

import labelsAPI, {
  getCustomLabels,
  listNamesFromSelectedLabels,
} from "services/entities/labels";
import mdmAPI, { isDDMProfile } from "services/entities/mdm";
import useGitOpsMode from "hooks/useGitOpsMode";

import Button from "components/buttons/Button";
import DataError from "components/DataError";
import FileUploader from "components/FileUploader";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Modal from "components/Modal";
import Spinner from "components/Spinner";
import {
  TargetLabelSelector,
  ILabelConfig,
  LabelTargetMode,
  TargetType,
} from "components/TargetLabelSelector";

import {
  generateCustomTargetLabelKey,
  getErrorMessage,
  IParseFileResult,
  parseFile,
} from "../ProfileUploader/helpers";

const baseClass = "edit-profile-modal";

export const getAcceptedExtensions = (profile: IMdmProfile) => {
  if (isDDMProfile(profile)) {
    return [".json"];
  }
  switch (profile.platform) {
    case "windows":
      return [".xml"];
    case "android":
      return [".json"];
    case "darwin":
    case "ios":
    case "ipados":
      // .xml is a valid mobileconfig: a profile is a bare-XML plist
      return [".mobileconfig", ".xml"];
    default:
      // unknown platform: accept nothing rather than guess
      return [];
  }
};

export const getProfileFileExtension = (profile: IMdmProfile) => {
  if (isDDMProfile(profile)) {
    return ".json";
  }
  switch (profile.platform) {
    case "windows":
      return ".xml";
    case "android":
      return ".json";
    case "darwin":
    case "ios":
    case "ipados":
      return ".mobileconfig";
    default:
      return "";
  }
};

const labelsToSelection = (labels?: IProfileLabel[]) =>
  (labels ?? []).reduce<Record<string, boolean>>((selection, label) => {
    selection[label.name] = true;
    return selection;
  }, {});

interface IEditProfileModalProps {
  profile: IMdmProfile;
  currentTeamId: number;
  isPremiumTier: boolean;
  /** called after a successful update; the caller is expected to refetch and
   * close the modal. */
  onUpdate: () => void;
  onCancel: () => void;
}

const EditProfileModal = ({
  profile,
  currentTeamId,
  isPremiumTier,
  onUpdate,
  onCancel,
}: IEditProfileModalProps) => {
  const { gitOpsModeEnabled } = useGitOpsMode();

  const initialIncludeLabels =
    profile.labels_include_all ?? profile.labels_include_any;
  const initialExcludeLabels = profile.labels_exclude_any;
  const hasCustomTarget =
    !!initialIncludeLabels?.length || !!initialExcludeLabels?.length;

  const [isUpdating, setIsUpdating] = useState(false);
  const [newFileDetails, setNewFileDetails] = useState<IParseFileResult | null>(
    null
  );
  const [selectedTargetType, setSelectedTargetType] = useState<TargetType>(
    hasCustomTarget ? "Custom" : "All hosts"
  );
  const [
    selectedLabelIncludeMode,
    setSelectedLabelIncludeMode,
  ] = useState<LabelTargetMode>(
    profile.labels_include_all?.length ? "all" : "any"
  );
  const [selectedIncludeLabels, setSelectedIncludeLabels] = useState(() =>
    labelsToSelection(initialIncludeLabels)
  );
  const [selectedExcludeLabels, setSelectedExcludeLabels] = useState(() =>
    labelsToSelection(initialExcludeLabels)
  );

  const fileRef = useRef<File | null>(null);

  const {
    data: labels,
    isLoading: isLoadingLabels,
    isFetching: isFetchingLabels,
    isError: isErrorLabels,
  } = useQuery<ILabelSummary[], Error>(
    ["custom_labels", currentTeamId],
    () =>
      labelsAPI
        .summary(currentTeamId)
        .then((res) => getCustomLabels(res.labels)),
    {
      enabled: isPremiumTier,
      refetchOnWindowFocus: false,
      retry: false,
      staleTime: 10000,
    }
  );

  const acceptedExtensions = getAcceptedExtensions(profile);

  const onFileSelected = async (files: FileList | null) => {
    if (!files || files.length === 0) {
      return;
    }
    const file = files[0];

    try {
      const details = await parseFile(file);
      if (!acceptedExtensions.includes(`.${details.ext}`)) {
        throw new Error(`Invalid file type: ${details.ext}`);
      }
      fileRef.current = file;
      setNewFileDetails(details);
    } catch (e) {
      notify.error("Invalid file type", { response: e });
    }
  };

  const onUpdateProfile = async () => {
    setIsUpdating(true);
    try {
      // labels use replace semantics on the API, so always submit the full
      // current label selection even when only the contents changed.
      const labelKey = generateCustomTargetLabelKey({
        targetType: selectedTargetType,
        includeMode: selectedLabelIncludeMode,
        includeLabels: selectedIncludeLabels,
        excludeLabels: selectedExcludeLabels,
      });
      await mdmAPI.updateProfile({
        profileUUID: profile.profile_uuid,
        profile: fileRef.current ?? undefined,
        ...labelKey,
      });
      notify.success("Successfully updated profile.");
      onUpdate();
    } catch (e) {
      notify.error(getErrorMessage(e as AxiosResponse<IApiError>, "edit"), {
        response: e,
      });
    } finally {
      setIsUpdating(false);
    }
  };

  const includeTab: ILabelConfig = {
    selectedLabels: selectedIncludeLabels,
    onSelectLabel: ({ name, value }) =>
      setSelectedIncludeLabels((prev) => ({ ...prev, [name]: value })),
    showModeToggle: true,
    mode: selectedLabelIncludeMode,
    onSelectMode: setSelectedLabelIncludeMode,
    anyTooltip: (
      <>
        Profile will be applied to hosts that{" "}
        <em>
          <b>have any</b>
        </em>{" "}
        of these labels.
      </>
    ),
    allTooltip: (
      <>
        Profile will be applied to hosts that{" "}
        <em>
          <b>have all</b>
        </em>{" "}
        of these labels.
      </>
    ),
  };

  const excludeTab: ILabelConfig = {
    selectedLabels: selectedExcludeLabels,
    onSelectLabel: ({ name, value }) =>
      setSelectedExcludeLabels((prev) => ({ ...prev, [name]: value })),
  };

  const hasSelectedLabels =
    listNamesFromSelectedLabels(selectedIncludeLabels).length > 0 ||
    listNamesFromSelectedLabels(selectedExcludeLabels).length > 0;

  return (
    <Modal className={baseClass} title="Edit profile" onExit={onCancel}>
      {isPremiumTier && isLoadingLabels && <Spinner />}
      {isPremiumTier && !isLoadingLabels && isErrorLabels && <DataError />}
      {(!isPremiumTier || (!isLoadingLabels && !isErrorLabels)) && (
        <div className={`${baseClass}__modal-content-wrap`}>
          <FileUploader
            canEdit
            graphicName="file-configuration-profile"
            accept={acceptedExtensions.join(",")}
            message={acceptedExtensions.join(", ")}
            onFileUpload={onFileSelected}
            fileDetails={{
              name: newFileDetails ? newFileDetails.name : profile.name,
              description: newFileDetails
                ? `.${newFileDetails.ext}`
                : getProfileFileExtension(profile),
            }}
            gitopsCompatible
            gitOpsModeEnabled={gitOpsModeEnabled}
          />
          {isPremiumTier && (
            <GitOpsModeTooltipWrapper
              isInputField
              renderChildren={(disableChildren) => (
                <div className={`form-field ${baseClass}__target`}>
                  <div className="form-field__label">Target</div>
                  <TargetLabelSelector
                    selectedTargetType={selectedTargetType}
                    onSelectTargetType={setSelectedTargetType}
                    labels={labels || []}
                    includeConfig={includeTab}
                    excludeConfig={excludeTab}
                    isLoadingLabels={isFetchingLabels}
                    isErrorLabels={isErrorLabels}
                    emptyStateDescription="Add a label to target your configuration profile."
                    onAddLabel={() => {
                      window.location.href = PATHS.LABEL_NEW_DYNAMIC;
                    }}
                    disableOptions={!!disableChildren}
                  />
                </div>
              )}
            />
          )}
          <div className={`${baseClass}__button-wrap`}>
            <Button variant="secondary" onClick={onCancel}>
              Cancel
            </Button>
            <GitOpsModeTooltipWrapper
              renderChildren={(disableChildren) => (
                <Button
                  className={`${baseClass}__update-profile-button`}
                  onClick={onUpdateProfile}
                  isLoading={isUpdating}
                  disabled={
                    disableChildren ||
                    isUpdating ||
                    (selectedTargetType === "Custom" && !hasSelectedLabels)
                  }
                >
                  Update profile
                </Button>
              )}
            />
          </div>
        </div>
      )}
    </Modal>
  );
};

export default EditProfileModal;
