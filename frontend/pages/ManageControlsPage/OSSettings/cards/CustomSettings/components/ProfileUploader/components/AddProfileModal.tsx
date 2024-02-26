import React, { useCallback, useContext, useRef, useState } from "react";
import { useQuery } from "react-query";
import { AxiosResponse } from "axios";

import { NotificationContext } from "context/notification";

import { IApiError } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";

import labelsAPI from "services/entities/labels";
import mdmAPI from "services/entities/mdm";

import Button from "components/buttons/Button";
import Card from "components/Card";
import Checkbox from "components/forms/fields/Checkbox";
import DataError from "components/DataError";
import Icon from "components/Icon";
import Modal from "components/Modal";
import Radio from "components/forms/fields/Radio";
import Spinner from "components/Spinner";

import ProfileGraphic from "./AddProfileGraphic";

import {
  DEFAULT_ERROR_MESSAGE,
  getErrorMessage,
  parseFile,
  listNamesFromSelectedLabels,
} from "../helpers";

const FileChooser = ({
  baseClass,
  isLoading,
  onFileOpen,
}: {
  baseClass: string;
  isLoading: boolean;
  onFileOpen: (files: FileList | null) => void;
}) => (
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
      accept=".mobileconfig,application/x-apple-aspen-config,.xml"
      id="upload-profile"
      type="file"
      onChange={(e) => {
        onFileOpen(e.target.files);
      }}
    />
  </div>
);

const FileDetails = ({
  baseClass,
  details: { name, platform },
}: {
  baseClass: string;
  details: {
    name: string;
    platform: string;
  };
}) => (
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

const TargetChooser = ({
  baseClass,
  selectedTarget,
  setSelectedTarget,
}: {
  baseClass: string;
  selectedTarget: string;
  setSelectedTarget: React.Dispatch<React.SetStateAction<string>>;
}) => {
  return (
    <div className={`form-field`}>
      <div className="form-field__label">Target</div>
      <Radio
        className={`${baseClass}__radio-input`}
        label="All hosts"
        id="all-hosts-target-radio-btn"
        checked={selectedTarget === "All hosts"}
        value="All hosts"
        name="all-hosts-target"
        onChange={setSelectedTarget}
      />
      <Radio
        className={`${baseClass}__radio-input`}
        label="Custom"
        id="custom-target-radio-btn"
        checked={selectedTarget === "Custom"}
        value="Custom"
        name="custom-target"
        onChange={setSelectedTarget}
      />
    </div>
  );
};

const LabelChooser = ({
  baseClass,
  isError,
  isLoading,
  labels,
  selectedLabels,
  setSelectedLabels,
}: {
  baseClass: string;
  isError: boolean;
  isLoading: boolean;
  labels: ILabelSummary[];
  selectedLabels: Record<string, boolean>;
  setSelectedLabels: React.Dispatch<
    React.SetStateAction<Record<string, boolean>>
  >;
}) => {
  const updateSelectedLabels = useCallback(
    ({ name, value }: { name: string; value: boolean }) => {
      setSelectedLabels((prevItems) => ({ ...prevItems, [name]: value }));
    },
    [setSelectedLabels]
  );
  return (
    <>
      <div className={`${baseClass}__description`}>
        Profile will only be applied to hosts that have all these labels:
      </div>
      <div className={`${baseClass}__checkboxes`}>
        {isLoading && <Spinner centered={false} />}
        {!isLoading && isError && <DataError />}
        {!isLoading && !isError && !labels.length && (
          <div className={`${baseClass}__no-labels`}>
            <b>No labels exist in Fleet</b>
            <span>Add labels to target specific hosts.</span>
          </div>
        )}
        {!isLoading &&
          !isError &&
          !!labels.length &&
          labels.map((label) => {
            return (
              <div className={`${baseClass}__label`} key={label.name}>
                <Checkbox
                  className={`${baseClass}__checkbox`}
                  name={label.name}
                  value={!!selectedLabels[label.name]}
                  onChange={updateSelectedLabels}
                  parseTarget
                />
                <div className={`${baseClass}__label-name`}>{label.name}</div>
              </div>
            );
          })}
      </div>
    </>
  );
};

interface IAddProfileModalProps {
  baseClass: string;
  currentTeamId: number;
  isPremiumTier: boolean;
  onUpload: () => void;
  setShowModal: React.Dispatch<React.SetStateAction<boolean>>;
}

const AddProfileModal = ({
  baseClass,
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
  const [selectedTarget, setSelectedTarget] = useState("All hosts"); // "All hosts" | "Custom"
  const [selectedLabels, setSelectedLabels] = useState<Record<string, boolean>>(
    {}
  );

  const fileRef = useRef<File | null>(null);

  // NOTE: labels are not automatically refetched in the current implementation
  const {
    data: labels,
    isLoading: isLoadingLabels,
    isFetching: isFetchingLabels,
    isError: isErrorLabels,
    // refetch: refetchLabels,
  } = useQuery<ILabelSummary[], Error>(
    ["custom_labels"], // NOTE: consider adding selectedTarget to the queryKey to refetch labels when target changes
    () =>
      labelsAPI
        .summary()
        .then((res) => res.labels.filter((l) => l.label_type !== "builtin")),

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
      await mdmAPI.uploadProfile({
        file,
        teamId: currentTeamId,
        labels: listNamesFromSelectedLabels(selectedLabels),
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

  return (
    <Modal title="Add profile" onExit={onDone}>
      <>
        {isPremiumTier && isLoadingLabels && <Spinner />}
        {isPremiumTier && !isLoadingLabels && isErrorLabels && <DataError />}
        {(!isPremiumTier || (!isLoadingLabels && !isErrorLabels)) && (
          <div className={`${baseClass}__modal-content-wrap`}>
            <Card color="gray" className={`${baseClass}__file`}>
              {!fileDetails ? (
                <FileChooser
                  baseClass={baseClass}
                  isLoading={isLoading}
                  onFileOpen={onFileOpen}
                />
              ) : (
                <FileDetails baseClass={baseClass} details={fileDetails} />
              )}
            </Card>
            {isPremiumTier && (
              <div className={`${baseClass}__target`}>
                <TargetChooser
                  baseClass={baseClass}
                  selectedTarget={selectedTarget}
                  setSelectedTarget={setSelectedTarget}
                />
                {selectedTarget === "Custom" && (
                  <LabelChooser
                    baseClass={baseClass}
                    isError={isErrorLabels}
                    isLoading={isFetchingLabels}
                    labels={labels || []}
                    selectedLabels={selectedLabels}
                    setSelectedLabels={setSelectedLabels}
                  />
                )}
              </div>
            )}
            <div className={`${baseClass}__button-wrap`}>
              <Button
                className={`${baseClass}__add-profile-button`}
                variant="brand"
                onClick={onFileUpload}
                isLoading={isLoading}
                disabled={
                  // TODO: consider adding tooltip to explain why button is disabled
                  (selectedTarget === "Custom" &&
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
