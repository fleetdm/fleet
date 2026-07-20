import React, { useCallback, useRef, useState } from "react";
import { useQuery } from "react-query";
import { AxiosResponse } from "axios";

import PATHS from "router/paths";
import { notify } from "components/ToastNotification";

import { IApiError } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";

import labelsAPI, {
  getCustomLabels,
  listNamesFromSelectedLabels,
} from "services/entities/labels";
import mdmAPI from "services/entities/mdm";

import Button from "components/buttons/Button";
import Card from "components/Card";
import DataError from "components/DataError";
import Icon from "components/Icon";
import Modal from "components/Modal";
import Spinner from "components/Spinner";
import {
  TargetLabelSelector,
  ILabelConfig,
  LabelTargetMode,
  TargetType,
} from "components/TargetLabelSelector";
import ProfileGraphic from "../ProfileGraphic";

import {
  DEFAULT_ERROR_MESSAGE,
  getErrorMessage,
  IParseFileResult,
  parseFile,
} from "../../helpers";
import generateCustomTargetLabelKey from "./helpers";

const baseClass = "add-profile-modal";

interface IFileChooserProps {
  isLoading: boolean;
  onFileOpen: (files: FileList | null) => void;
}

/** TODO: Legacy component, should be replaced with newer FileUploader */
const FileChooser = ({ isLoading, onFileOpen }: IFileChooserProps) => (
  <div className={`${baseClass}__file-chooser`}>
    <ProfileGraphic
      baseClass={baseClass}
      title="Upload configuration profile"
      message={
        <>
          .mobileconfig and .json for macOS, iOS, and iPadOS.
          <br />
          .json for Android.
          <br />
          .xml for Windows.
        </>
      }
    />
    <Button
      className={`${baseClass}__upload-button`}
      variant="brand-inverse-icon"
      isLoading={isLoading}
    >
      <label htmlFor="upload-profile">
        <span className={`${baseClass}__file-chooser--button-wrap`}>
          Choose file <Icon name="upload" color="core-fleet-green" />
        </span>
      </label>
    </Button>
    <input
      accept=".json,.mobileconfig,application/x-apple-aspen-config,.xml"
      id="upload-profile"
      type="file"
      onChange={(e) => {
        onFileOpen(e.target.files);
      }}
    />
  </div>
);

interface IFileDetailsProps {
  details: IParseFileResult;
}

// TODO: if we reuse this one more time, we should consider moving this
// into FileUploader as a default preview. Currently we have this in
// AddPackageForm.tsx and here.
const FileDetails = ({ details: { name, ext } }: IFileDetailsProps) => (
  <div className={`${baseClass}__selected-file`}>
    <ProfileGraphic baseClass={baseClass} />
    <div className={`${baseClass}__selected-file--details`}>
      <div className={`${baseClass}__selected-file--details--name`}>{name}</div>
      <div className={`${baseClass}__selected-file--details--platform`}>
        .{ext}
      </div>
    </div>
  </div>
);

interface IAddProfileModalProps {
  currentTeamId: number;
  isPremiumTier: boolean;
  onUpload: () => void;
  setShowModal: React.Dispatch<React.SetStateAction<boolean>>;
}

const AddProfileModal = ({
  currentTeamId,
  isPremiumTier,
  onUpload,
  setShowModal,
}: IAddProfileModalProps) => {
  const [isLoading, setIsLoading] = useState(false);
  const [fileDetails, setFileDetails] = useState<IParseFileResult | null>(null);
  const [selectedTargetType, setSelectedTargetType] = useState<TargetType>(
    "All hosts"
  );
  const [
    selectedLabelIncludeMode,
    setSelectedLabelIncludeMode,
  ] = useState<LabelTargetMode>("any");
  const [selectedIncludeLabels, setSelectedIncludeLabels] = useState<
    Record<string, boolean>
  >({});
  const [selectedExcludeLabels, setSelectedExcludeLabels] = useState<
    Record<string, boolean>
  >({});

  const fileRef = useRef<File | null>(null);

  const {
    data: labels,
    isLoading: isLoadingLabels,
    isFetching: isFetchingLabels,
    isError: isErrorLabels,
  } = useQuery<ILabelSummary[], Error>(
    ["custom_labels"],
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

  const onDone = useCallback(() => {
    fileRef.current = null;
    setFileDetails(null);
    setSelectedIncludeLabels({});
    setSelectedExcludeLabels({});
    setShowModal(false);
  }, [fileRef, setShowModal]);

  const onFileUpload = async () => {
    if (!fileRef.current) {
      notify.error(DEFAULT_ERROR_MESSAGE);
      return;
    }
    const file = fileRef.current;

    setIsLoading(true);
    try {
      const labelKey = generateCustomTargetLabelKey({
        targetType: selectedTargetType,
        includeMode: selectedLabelIncludeMode,
        includeLabels: selectedIncludeLabels,
        excludeLabels: selectedExcludeLabels,
      });
      await mdmAPI.uploadProfile({
        file,
        teamId: currentTeamId,
        ...labelKey,
      });
      notify.success("Successfully uploaded.");
      onUpload();
    } catch (e) {
      notify.error(getErrorMessage(e as AxiosResponse<IApiError>), {
        response: e,
      });
    } finally {
      setIsLoading(false);
      onDone();
    }
  };

  const onFileOpen = async (files: FileList | null) => {
    if (!files || files.length === 0) {
      setIsLoading(false);
      return;
    }

    setIsLoading(true);
    const file = files[0];
    fileRef.current = file;

    try {
      const details = await parseFile(file);
      setFileDetails(details);
    } catch (e) {
      notify.error("Invalid file type", { response: e });
    } finally {
      setIsLoading(false);
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
    <Modal title="Add profile" onExit={onDone}>
      {isPremiumTier && isLoadingLabels && <Spinner />}
      {isPremiumTier && !isLoadingLabels && isErrorLabels && <DataError />}
      {(!isPremiumTier || (!isLoadingLabels && !isErrorLabels)) && (
        <div className={`${baseClass}__modal-content-wrap`}>
          <Card color="grey" className={`${baseClass}__file`}>
            {!fileDetails ? (
              <FileChooser isLoading={isLoading} onFileOpen={onFileOpen} />
            ) : (
              <FileDetails details={fileDetails} />
            )}
          </Card>
          {isPremiumTier && (
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
              />
            </div>
          )}
          <div className={`${baseClass}__button-wrap`}>
            <Button variant="secondary" onClick={onDone}>
              Cancel
            </Button>
            <Button
              className={`${baseClass}__add-profile-button`}
              onClick={onFileUpload}
              isLoading={isLoading}
              disabled={
                (selectedTargetType === "Custom" && !hasSelectedLabels) ||
                !fileDetails
              }
            >
              Add profile
            </Button>
          </div>
        </div>
      )}
    </Modal>
  );
};

export default AddProfileModal;
