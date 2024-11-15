import React, { useCallback, useContext, useRef, useState } from "react";
import { useQuery } from "react-query";
import { AxiosResponse } from "axios";

import { NotificationContext } from "context/notification";

import { IApiError } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";

import labelsAPI, { getCustomLabels } from "services/entities/labels";
import mdmAPI from "services/entities/mdm";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Button from "components/buttons/Button";
import Card from "components/Card";
import Checkbox from "components/forms/fields/Checkbox";
import DataError from "components/DataError";
import Icon from "components/Icon";
import Modal from "components/Modal";
import Radio from "components/forms/fields/Radio";
import Spinner from "components/Spinner";

import ProfileGraphic from "../AddProfileGraphic";

import {
  DEFAULT_ERROR_MESSAGE,
  getErrorMessage,
  parseFile,
} from "../../helpers";
import {
  CUSTOM_TARGET_OPTIONS,
  CustomTargetOption,
  generateLabelKey,
  getDescriptionText,
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

interface ITargetChooserProps {
  selectedTarget: string;
  setSelectedTarget: React.Dispatch<React.SetStateAction<string>>;
}

const TargetChooser = ({
  selectedTarget,
  setSelectedTarget,
}: ITargetChooserProps) => {
  return (
    <div className={`form-field`}>
      <div className="form-field__label">Target</div>
      <Radio
        className={`${baseClass}__radio-input`}
        label="All hosts"
        id="all-hosts-target-radio-btn"
        checked={selectedTarget === "All hosts"}
        value="All hosts"
        name="target-type"
        onChange={setSelectedTarget}
      />
      <Radio
        className={`${baseClass}__radio-input`}
        label="Custom"
        id="custom-target-radio-btn"
        checked={selectedTarget === "Custom"}
        value="Custom"
        name="target-type"
        onChange={setSelectedTarget}
      />
    </div>
  );
};

interface ILabelChooserProps {
  isError: boolean;
  isLoading: boolean;
  labels: ILabelSummary[];
  selectedLabels: Record<string, boolean>;
  customTargetOption: CustomTargetOption;
  setSelectedLabels: React.Dispatch<
    React.SetStateAction<Record<string, boolean>>
  >;
  onSelectCustomTargetOption: (val: CustomTargetOption) => void;
}

const LabelChooser = ({
  isError,
  isLoading,
  labels,
  selectedLabels,
  customTargetOption,
  setSelectedLabels,
  onSelectCustomTargetOption,
}: ILabelChooserProps) => {
  const updateSelectedLabels = useCallback(
    ({ name, value }: { name: string; value: boolean }) => {
      setSelectedLabels((prevItems) => ({ ...prevItems, [name]: value }));
    },
    [setSelectedLabels]
  );

  const renderLabels = () => {
    if (isLoading) {
      return <Spinner centered={false} />;
    }

    if (isError) {
      return <DataError />;
    }

    if (!labels.length) {
      return (
        <div className={`${baseClass}__no-labels`}>
          <b>No labels exist in Fleet</b>
          <span>Add labels to target specific hosts.</span>
        </div>
      );
    }

    return labels.map((label) => {
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
    });
  };

  return (
    <div className={`${baseClass}__custom-label-chooser`}>
      <Dropdown
        value={customTargetOption}
        options={CUSTOM_TARGET_OPTIONS}
        searchable={false}
        onChange={onSelectCustomTargetOption}
      />
      <div className={`${baseClass}__description`}>
        {getDescriptionText(customTargetOption)}
      </div>
      <div className={`${baseClass}__checkboxes`}>{renderLabels()}</div>
    </div>
  );
};

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
  const [selectedTarget, setSelectedTarget] = useState("All hosts"); // "All hosts" | "Custom"
  const [selectedLabels, setSelectedLabels] = useState<Record<string, boolean>>(
    {}
  );
  const [
    customTargetOption,
    setCustomTargetOption,
  ] = useState<CustomTargetOption>("labelsIncludeAll");

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
        selectedTarget,
        customTargetOption,
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

  const onSelectCustomTargetOption = (val: CustomTargetOption) => {
    setCustomTargetOption(val);
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
              <div className={`${baseClass}__target`}>
                <TargetChooser
                  selectedTarget={selectedTarget}
                  setSelectedTarget={setSelectedTarget}
                />
                {selectedTarget === "Custom" && (
                  <LabelChooser
                    customTargetOption={customTargetOption}
                    isError={isErrorLabels}
                    isLoading={isFetchingLabels}
                    labels={labels || []}
                    selectedLabels={selectedLabels}
                    setSelectedLabels={setSelectedLabels}
                    onSelectCustomTargetOption={onSelectCustomTargetOption}
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
