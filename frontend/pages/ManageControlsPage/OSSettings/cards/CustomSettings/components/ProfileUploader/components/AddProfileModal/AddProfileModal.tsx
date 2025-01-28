import React, { useCallback, useContext, useRef, useState } from "react";
import { useQuery } from "react-query";
import { AxiosResponse } from "axios";

import { NotificationContext } from "context/notification";

import { IApiError } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";

import labelsAPI, { getCustomLabels } from "services/entities/labels";
import mdmAPI from "services/entities/mdm";

// @ts-ignore
import Button from "components/buttons/Button";
import Card from "components/Card";
import DataError from "components/DataError";
import Icon from "components/Icon";
import Modal from "components/Modal";
import Spinner from "components/Spinner";
import TargetLabelSelector from "components/TargetLabelSelector";
import ProfileGraphic from "../AddProfileGraphic";

import {
  DEFAULT_ERROR_MESSAGE,
  getErrorMessage,
  parseFile,
} from "../../helpers";
import {
  CUSTOM_TARGET_OPTIONS,
  generateLabelKey,
  listNamesFromSelectedLabels,
} from "./helpers";

const baseClass = "add-profile-modal";

interface IFileChooserProps {
  isLoading: boolean;
  onFileOpen: (files: FileList | null) => void;
}

const FileChooser = ({ isLoading, onFileOpen }: IFileChooserProps) => (
  <div className={`${baseClass}__file-chooser`}>
    <ProfileGraphic baseClass={baseClass} showMessage />
    <Button
      className={`${baseClass}__upload-button`}
      variant="text-icon"
      isLoading={isLoading}
    >
      <label htmlFor="upload-profile">
        <span className={`${baseClass}__file-chooser--button-wrap`}>
          <Icon name="upload" />
          Choose file
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
  details: {
    name: string;
    platform: string;
  };
}

// TODO: if we reuse this one more time, we should consider moving this
// into FileUploader as a default preview. Currently we have this in
// AddPackageForm.tsx and here.
const FileDetails = ({ details: { name, platform } }: IFileDetailsProps) => (
  <div className={`${baseClass}__selected-file`}>
    <ProfileGraphic baseClass={baseClass} />
    <div className={`${baseClass}__selected-file--details`}>
      <div className={`${baseClass}__selected-file--details--name`}>{name}</div>
      <div className={`${baseClass}__selected-file--details--platform`}>
        {platform}
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
  const { renderFlash } = useContext(NotificationContext);

  const [isLoading, setIsLoading] = useState(false);
  const [fileDetails, setFileDetails] = useState<{
    name: string;
    platform: string;
  } | null>(null);
  const [selectedTargetType, setSelectedTargetType] = useState("All hosts");
  const [selectedLabels, setSelectedLabels] = useState<Record<string, boolean>>(
    {}
  );
  const [selectedCustomTarget, setSelectedCustomTarget] = useState(
    "labelsIncludeAll"
  );

  const fileRef = useRef<File | null>(null);

  const {
    data: labels,
    isLoading: isLoadingLabels,
    isFetching: isFetchingLabels,
    isError: isErrorLabels,
  } = useQuery<ILabelSummary[], Error>(
    ["custom_labels"],
    () => labelsAPI.summary().then((res) => getCustomLabels(res.labels)),
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
    setSelectedLabels({});
    setShowModal(false);
  }, [fileRef, setShowModal]);

  const onFileUpload = async () => {
    if (!fileRef.current) {
      renderFlash("error", DEFAULT_ERROR_MESSAGE);
      return;
    }
    const file = fileRef.current;

    setIsLoading(true);
    try {
      const labelKey = generateLabelKey(
        selectedTargetType,
        selectedCustomTarget,
        selectedLabels
      );
      await mdmAPI.uploadProfile({
        file,
        teamId: currentTeamId,
        ...labelKey,
      });
      renderFlash("success", "Successfully uploaded!");
      onUpload();
    } catch (e) {
      // TODO: cleanup this error handling
      renderFlash("error", getErrorMessage(e as AxiosResponse<IApiError>));
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
      const [name, platform] = await parseFile(file);
      setFileDetails({ name, platform });
    } catch (e) {
      renderFlash("error", "Invalid file type");
    } finally {
      setIsLoading(false);
    }
  };

  const onSelectTargetType = (val: string) => {
    setSelectedTargetType(val);
  };

  const onSelectCustomTargetOption = (val: string) => {
    setSelectedCustomTarget(val);
  };

  const onSelectLabel = ({ name, value }: { name: string; value: boolean }) => {
    setSelectedLabels((prevItems) => ({ ...prevItems, [name]: value }));
  };

  return (
    <Modal title="Add profile" onExit={onDone}>
      <>
        {isPremiumTier && isLoadingLabels && <Spinner />}
        {isPremiumTier && !isLoadingLabels && isErrorLabels && <DataError />}
        {(!isPremiumTier || (!isLoadingLabels && !isErrorLabels)) && (
          <div className={`${baseClass}__modal-content-wrap`}>
            <Card color="gray" className={`${baseClass}__file`}>
              {!fileDetails ? (
                <FileChooser isLoading={isLoading} onFileOpen={onFileOpen} />
              ) : (
                <FileDetails details={fileDetails} />
              )}
            </Card>
            {isPremiumTier && (
              <TargetLabelSelector
                selectedTargetType={selectedTargetType}
                selectedCustomTarget={selectedCustomTarget}
                selectedLabels={selectedLabels}
                customTargetOptions={CUSTOM_TARGET_OPTIONS}
                className={`${baseClass}__target`}
                onSelectTargetType={onSelectTargetType}
                onSelectCustomTarget={onSelectCustomTargetOption}
                onSelectLabel={onSelectLabel}
                isErrorLabels={isErrorLabels}
                isLoadingLabels={isFetchingLabels || isLoadingLabels}
                labels={labels || []}
              />
            )}
            <div className={`${baseClass}__button-wrap`}>
              <Button
                className={`${baseClass}__add-profile-button`}
                variant="brand"
                onClick={onFileUpload}
                isLoading={isLoading}
                disabled={
                  (selectedTargetType === "Custom" &&
                    !listNamesFromSelectedLabels(selectedLabels).length) ||
                  !fileDetails
                }
              >
                Add profile
              </Button>
            </div>
          </div>
        )}
      </>
    </Modal>
  );
};

export default AddProfileModal;
