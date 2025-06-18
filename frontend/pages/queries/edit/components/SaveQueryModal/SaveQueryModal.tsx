import React, { useState, useEffect, useContext } from "react";
import { useQuery } from "react-query";

import { size } from "lodash";

import { AppContext } from "context/app";

import useDeepEffect from "hooks/useDeepEffect";
import { IPlatformSelector } from "hooks/usePlatformSelector";

import {
  FREQUENCY_DROPDOWN_OPTIONS,
  LOGGING_TYPE_OPTIONS,
  MIN_OSQUERY_VERSION_OPTIONS,
  DEFAULT_USE_QUERY_OPTIONS,
} from "utilities/constants";

import { CommaSeparatedPlatformString } from "interfaces/platform";
import {
  ICreateQueryRequestBody,
  ISchedulableQuery,
  QueryLoggingOption,
} from "interfaces/schedulable_query";

import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Slider from "components/forms/fields/Slider";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import RevealButton from "components/buttons/RevealButton";
import LogDestinationIndicator from "components/LogDestinationIndicator";
import TargetLabelSelector from "components/TargetLabelSelector";
import labelsAPI, {
  getCustomLabels,
  ILabelsSummaryResponse,
} from "services/entities/labels";

import DiscardDataOption from "../DiscardDataOption";

const baseClass = "save-query-modal";
export interface ISaveQueryModalProps {
  queryValue: string;
  apiTeamIdForQuery?: number; // query will be global if omitted
  isLoading: boolean;
  saveQuery: (formData: ICreateQueryRequestBody) => void;
  toggleSaveQueryModal: () => void;
  backendValidators: { [key: string]: string };
  existingQuery?: ISchedulableQuery;
  queryReportsDisabled?: boolean;
  platformSelector: IPlatformSelector;
}

const validateQueryName = (name: string) => {
  const errors: { [key: string]: string } = {};

  if (!name) {
    errors.name = "Query name must be present";
  }

  const valid = !size(errors);
  return { valid, errors };
};

const SaveQueryModal = ({
  queryValue,
  apiTeamIdForQuery,
  isLoading,
  saveQuery,
  toggleSaveQueryModal,
  backendValidators,
  existingQuery,
  queryReportsDisabled,
  platformSelector,
}: ISaveQueryModalProps): JSX.Element => {
  const { config, isPremiumTier } = useContext(AppContext);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [selectedFrequency, setSelectedFrequency] = useState(
    existingQuery?.interval ?? 3600
  );
  const [
    selectedMinOsqueryVersionOptions,
    setSelectedMinOsqueryVersionOptions,
  ] = useState(existingQuery?.min_osquery_version ?? "");
  const [
    selectedLoggingType,
    setSelectedLoggingType,
  ] = useState<QueryLoggingOption>(existingQuery?.logging ?? "snapshot");
  const [observerCanRun, setObserverCanRun] = useState(false);
  const [automationsEnabled, setAutomationsEnabled] = useState(false);
  const [selectedTargetType, setSelectedTargetType] = useState("All hosts");
  const [selectedLabels, setSelectedLabels] = useState({});
  const [discardData, setDiscardData] = useState(false);
  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

  const {
    data: { labels } = { labels: [] },
    isFetching: isFetchingLabels,
  } = useQuery<ILabelsSummaryResponse, Error>(
    ["custom_labels"],
    () => labelsAPI.summary(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier,
      staleTime: 10000,
      select: (res) => ({ labels: getCustomLabels(res.labels) }),
    }
  );

  const onSelectLabel = ({
    name: labelName,
    value,
  }: {
    name: string;
    value: boolean;
  }) => {
    setSelectedLabels({
      ...selectedLabels,
      [labelName]: value,
    });
  };

  const toggleAdvancedOptions = () => {
    setShowAdvancedOptions(!showAdvancedOptions);
  };

  useDeepEffect(() => {
    if (name) {
      setErrors({});
    }
  }, [name]);

  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  // Disable saving if "Custom" targeting is selected, but no labels are selected.
  const canSave =
    platformSelector.isAnyPlatformSelected &&
    (selectedTargetType === "All hosts" ||
      Object.entries(selectedLabels).some(([, value]) => {
        return value;
      }));

  const onClickSaveQuery = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const trimmedName = name.trim();

    const { valid, errors: newErrors } = validateQueryName(trimmedName);
    setErrors({
      ...errors,
      ...newErrors,
    });
    setName(trimmedName);

    const newPlatformString = platformSelector
      .getSelectedPlatforms()
      .join(",") as CommaSeparatedPlatformString;

    if (valid) {
      saveQuery({
        // from modal fields
        name: trimmedName,
        description,
        interval: selectedFrequency,
        observer_can_run: observerCanRun,
        automations_enabled: automationsEnabled,
        discard_data: discardData,
        platform: newPlatformString,
        min_osquery_version: selectedMinOsqueryVersionOptions,
        logging: selectedLoggingType,
        // from previous New query page
        query: queryValue,
        // from doubly previous ManageQueriesPage
        team_id: apiTeamIdForQuery,
        labels_include_any:
          selectedTargetType === "Custom"
            ? Object.entries(selectedLabels)
                .filter(([, selected]) => selected)
                .map(([labelName]) => labelName)
            : [],
      });
    }
  };

  return (
    <Modal title="Save query" onExit={toggleSaveQueryModal}>
      <form
        onSubmit={onClickSaveQuery}
        className={baseClass}
        autoComplete="off"
      >
        <InputField
          name="name"
          onChange={(value: string) => setName(value)}
          onBlur={() => {
            setName(name.trim());
          }}
          value={name}
          error={errors.name}
          inputClassName={`${baseClass}__name`}
          label="Name"
          autofocus
          ignore1password
        />
        <InputField
          name="description"
          onChange={(value: string) => setDescription(value)}
          value={description}
          inputClassName={`${baseClass}__description`}
          label="Description"
          type="textarea"
          helpText="What information does your query reveal? (optional)"
        />
        <Dropdown
          searchable={false}
          options={FREQUENCY_DROPDOWN_OPTIONS}
          onChange={(value: number) => {
            setSelectedFrequency(value);
          }}
          placeholder="Every hour"
          value={selectedFrequency}
          label="Interval"
          wrapperClassName={`${baseClass}__form-field form-field--frequency`}
          helpText="This is how often your query collects data."
        />
        <Checkbox
          name="observerCanRun"
          onChange={setObserverCanRun}
          value={observerCanRun}
          wrapperClassName="observer-can-run-wrapper"
          helpText="Users with the Observer role will be able to run this query as a live query."
        >
          Observers can run
        </Checkbox>
        <Slider
          onChange={() => setAutomationsEnabled(!automationsEnabled)}
          value={automationsEnabled}
          activeText={
            <>
              Automations on
              {selectedFrequency === 0 && (
                <TooltipWrapper
                  tipContent={
                    <>
                      Automations and reporting will be paused <br />
                      for this query until an interval is set.
                    </>
                  }
                  position="right"
                  tipOffset={9}
                  showArrow
                  underline={false}
                >
                  <Icon name="warning" />
                </TooltipWrapper>
              )}
            </>
          }
          inactiveText="Automations off"
          helpText={
            <>
              Historical results will {!automationsEnabled ? "not " : ""}be sent
              to your log destination:{" "}
              <b>
                <LogDestinationIndicator
                  logDestination={config?.logging.result.plugin || ""}
                  filesystemDestination={
                    config?.logging.result.config.result_log_file
                  }
                  excludeTooltip
                />
              </b>
              .
            </>
          }
        />
        {platformSelector.render()}
        {isPremiumTier && (
          <TargetLabelSelector
            selectedTargetType={selectedTargetType}
            selectedLabels={selectedLabels}
            className={`${baseClass}__target`}
            onSelectTargetType={setSelectedTargetType}
            onSelectLabel={onSelectLabel}
            labels={labels || []}
            customHelpText={
              <span className="form-field__help-text">
                Query will target hosts that <b>have any</b> of these labels:
              </span>
            }
            suppressTitle
          />
        )}
        <RevealButton
          isShowing={showAdvancedOptions}
          className="advanced-options-toggle"
          hideText="Hide advanced options"
          showText="Show advanced options"
          caretPosition="after"
          onClick={toggleAdvancedOptions}
        />
        {showAdvancedOptions && (
          <>
            <Dropdown
              options={MIN_OSQUERY_VERSION_OPTIONS}
              onChange={setSelectedMinOsqueryVersionOptions}
              placeholder="Select"
              value={selectedMinOsqueryVersionOptions}
              label="Minimum osquery version"
              wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--osquer-vers`}
            />
            <Dropdown
              options={LOGGING_TYPE_OPTIONS}
              onChange={setSelectedLoggingType}
              placeholder="Select"
              value={selectedLoggingType}
              label="Logging"
              wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--logging`}
            />
            {queryReportsDisabled !== undefined && (
              <DiscardDataOption
                {...{
                  queryReportsDisabled,
                  selectedLoggingType,
                  discardData,
                  setDiscardData,
                  breakHelpText: true,
                }}
              />
            )}
          </>
        )}
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            className="save-query-loading"
            isLoading={isLoading || isFetchingLabels}
            disabled={!canSave}
          >
            Save
          </Button>
          <Button onClick={toggleSaveQueryModal} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default SaveQueryModal;
