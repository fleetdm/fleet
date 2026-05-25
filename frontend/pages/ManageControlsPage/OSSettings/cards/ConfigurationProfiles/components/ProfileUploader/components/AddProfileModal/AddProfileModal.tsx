import React, { useCallback, useContext, useRef, useState } from "react";
import { useQuery } from "react-query";
import { AxiosResponse } from "axios";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import PATHS from "router/paths";
import { NotificationContext } from "context/notification";

import { IApiError } from "interfaces/errors";
import { ILabelSummary } from "interfaces/label";

import labelsAPI, { getCustomLabels } from "services/entities/labels";
import mdmAPI from "services/entities/mdm";
import SearchField from "components/forms/fields/SearchField";

// @ts-ignore
import Button from "components/buttons/Button";
import Card from "components/Card";
// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import Icon from "components/Icon";
import Modal from "components/Modal";
// @ts-ignore
import Radio from "components/forms/fields/Radio";
import Spinner from "components/Spinner";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import ProfileGraphic from "../ProfileGraphic";

import {
  DEFAULT_ERROR_MESSAGE,
  getErrorMessage,
  IParseFileResult,
  parseFile,
} from "../../helpers";
import {
  generateLabelKey,
  listNamesFromSelectedLabels,
} from "./helpers";

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
  const { renderFlash } = useContext(NotificationContext);

  const [isLoading, setIsLoading] = useState(false);
  const [fileDetails, setFileDetails] = useState<IParseFileResult | null>(null);
  const [selectedTargetType, setSelectedTargetType] = useState("All hosts");
  const [selectedTabIndex, setSelectedTabIndex] = useState(0);
  const [selectedIncludeMode, setSelectedIncludeMode] = useState<"any" | "all">("any");
  const [selectedExcludeMode, setSelectedExcludeMode] = useState<"any" | "all">("any");
  const [selectedIncludeLabels, setSelectedIncludeLabels] = useState<
    Record<string, boolean>
  >({});
  const [selectedExcludeLabels, setSelectedExcludeLabels] = useState<
    Record<string, boolean>
  >({});
  const [includeSearchQuery, setIncludeSearchQuery] = useState("");
  const [excludeSearchQuery, setExcludeSearchQuery] = useState("");

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
    setIncludeSearchQuery("");
    setExcludeSearchQuery("");
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
        selectedIncludeMode,
        selectedIncludeLabels,
        selectedExcludeMode,
        selectedExcludeLabels
      );
      await mdmAPI.uploadProfile({
        file,
        teamId: currentTeamId,
        ...labelKey,
      });
      renderFlash("success", "Successfully uploaded.");
      onUpload();
    } catch (e) {
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
      const details = await parseFile(file);
      setFileDetails(details);
    } catch (e) {
      renderFlash("error", "Invalid file type");
    } finally {
      setIsLoading(false);
    }
  };

  const onSelectIncludeLabel = ({
    name,
    value,
  }: {
    name: string;
    value: boolean;
  }) => {
    setSelectedIncludeLabels((prev) => ({ ...prev, [name]: value }));
  };

  const onSelectExcludeLabel = ({
    name,
    value,
  }: {
    name: string;
    value: boolean;
  }) => {
    setSelectedExcludeLabels((prev) => ({ ...prev, [name]: value }));
  };

  const renderSelectedBadges = (
    selected: Record<string, boolean>,
    onChange: (arg: { name: string; value: boolean }) => void
  ) => {
    const selectedNames = listNamesFromSelectedLabels(selected);
    if (!selectedNames.length) return null;
    return (
      <div className={`${baseClass}__selected-badges`}>
        {selectedNames.map((name) => (
          <button
            key={name}
            className={`${baseClass}__selected-badge`}
            onClick={() => onChange({ name, value: false })}
          >
            <span>{name}</span>
            <Icon name="close" size="small" color="ui-fleet-black-75" />
          </button>
        ))}
      </div>
    );
  };

  const renderLabelCheckboxes = (
    filteredLabels: ILabelSummary[],
    selected: Record<string, boolean>,
    disabledLabels: Record<string, boolean>,
    onChange: (arg: { name: string; value: boolean }) => void
  ) => (
    <div className="target-label-selector__checkboxes">
      {filteredLabels.map((label) => (
        <div className="target-label-selector__label" key={label.name}>
          <Checkbox
            className="target-label-selector__checkbox"
            name={label.name}
            value={!!selected[label.name]}
            disabled={!!disabledLabels[label.name]}
            onChange={onChange}
            parseTarget
          >
            {label.name}
          </Checkbox>
        </div>
      ))}
    </div>
  );

  const renderCustomTarget = () => {
    if (isFetchingLabels || isLoadingLabels) {
      return <Spinner centered={false} />;
    }
    if (isErrorLabels) {
      return <DataError />;
    }
    if (!labels?.length) {
      return (
        <div className="target-label-selector__no-labels">
          <span>No labels</span>
          <CustomLink url={PATHS.LABEL_NEW_DYNAMIC} text="Add label" /> to
          target specific hosts.
        </div>
      );
    }

    const filteredIncludeLabels = (labels || []).filter((l) =>
      l.name.toLowerCase().includes(includeSearchQuery.toLowerCase())
    );
    const filteredExcludeLabels = (labels || []).filter((l) =>
      l.name.toLowerCase().includes(excludeSearchQuery.toLowerCase())
    );

    return (
      <TabNav secondary>
        <Tabs
          selectedIndex={selectedTabIndex}
          onSelect={setSelectedTabIndex}
        >
          <TabList>
            <Tab>
              <TabText>Include</TabText>
            </Tab>
            <Tab>
              <TabText>Exclude</TabText>
            </Tab>
          </TabList>
          <TabPanel>
            <Radio
              className="target-label-selector__radio-input"
              label="Any"
              id="include-any-radio"
              checked={selectedIncludeMode === "any"}
              value="any"
              name="include-mode"
              onChange={(val: string) => setSelectedIncludeMode(val as "any" | "all")}
            />
            <Radio
              className="target-label-selector__radio-input"
              label="All"
              id="include-all-radio"
              checked={selectedIncludeMode === "all"}
              value="all"
              name="include-mode"
              onChange={(val: string) => setSelectedIncludeMode(val as "any" | "all")}
            />
            <SearchField
              placeholder="Search labels"
              onChange={setIncludeSearchQuery}
            />
            {renderSelectedBadges(selectedIncludeLabels, onSelectIncludeLabel)}
            {renderLabelCheckboxes(filteredIncludeLabels, selectedIncludeLabels, selectedExcludeLabels, onSelectIncludeLabel)}
          </TabPanel>
          <TabPanel>
            <Radio
              className="target-label-selector__radio-input"
              label="Any"
              id="exclude-any-radio"
              checked={selectedExcludeMode === "any"}
              value="any"
              name="exclude-mode"
              onChange={(val: string) => setSelectedExcludeMode(val as "any" | "all")}
            />
            <Radio
              className="target-label-selector__radio-input"
              label="All"
              id="exclude-all-radio"
              checked={selectedExcludeMode === "all"}
              value="all"
              name="exclude-mode"
              onChange={(val: string) => setSelectedExcludeMode(val as "any" | "all")}
            />
            <SearchField
              placeholder="Search labels"
              onChange={setExcludeSearchQuery}
            />
            {renderSelectedBadges(selectedExcludeLabels, onSelectExcludeLabel)}
            {renderLabelCheckboxes(filteredExcludeLabels, selectedExcludeLabels, selectedIncludeLabels, onSelectExcludeLabel)}
          </TabPanel>
        </Tabs>
      </TabNav>
    );
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
            <div className={`target-label-selector form ${baseClass}__target`}>
              <div className="form-field">
                <div className="form-field__label">Target</div>
                <Radio
                  className="target-label-selector__radio-input"
                  label="All hosts"
                  id="all-hosts-target-radio-btn"
                  checked={selectedTargetType === "All hosts"}
                  value="All hosts"
                  name="target-type"
                  onChange={setSelectedTargetType}
                />
                <Radio
                  className="target-label-selector__radio-input"
                  label="Custom"
                  id="custom-target-radio-btn"
                  checked={selectedTargetType === "Custom"}
                  value="Custom"
                  name="target-type"
                  onChange={setSelectedTargetType}
                />
              </div>
              {selectedTargetType === "Custom" && renderCustomTarget()}
            </div>
          )}
          <div className={`${baseClass}__button-wrap`}>
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
