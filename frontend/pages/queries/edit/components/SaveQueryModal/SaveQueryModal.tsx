import React, { useState, useEffect, useCallback, useContext } from "react";
import { pull, size } from "lodash";

import { AppContext } from "context/app";

import useDeepEffect from "hooks/useDeepEffect";

import {
  FREQUENCY_DROPDOWN_OPTIONS,
  LOGGING_TYPE_OPTIONS,
  MIN_OSQUERY_VERSION_OPTIONS,
  SCHEDULE_PLATFORM_DROPDOWN_OPTIONS,
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
}: ISaveQueryModalProps): JSX.Element => {
  const { config } = useContext(AppContext);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [selectedFrequency, setSelectedFrequency] = useState(
    existingQuery?.interval ?? 3600
  );
  const [
    selectedPlatformOptions,
    setSelectedPlatformOptions,
  ] = useState<CommaSeparatedPlatformString>(existingQuery?.platform ?? "");
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
  const [discardData, setDiscardData] = useState(false);
  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

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

  const onClickSaveQuery = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const trimmedName = name.trim();

    const { valid, errors: newErrors } = validateQueryName(trimmedName);
    setErrors({
      ...errors,
      ...newErrors,
    });
    setName(trimmedName);

    if (valid) {
      saveQuery({
        // from modal fields
        name: trimmedName,
        description,
        interval: selectedFrequency,
        observer_can_run: observerCanRun,
        automations_enabled: automationsEnabled,
        discard_data: discardData,
        platform: selectedPlatformOptions,
        min_osquery_version: selectedMinOsqueryVersionOptions,
        logging: selectedLoggingType,
        // from previous New query page
        query: queryValue,
        // from doubly previous ManageQueriesPage
        team_id: apiTeamIdForQuery,
      });
    }
  };

  const onChangeSelectPlatformOptions = useCallback(
    (values: string) => {
      const valArray = values.split(",");

      // Remove All if another OS is chosen
      // else if Remove OS if All is chosen
      if (valArray.indexOf("") === 0 && valArray.length > 1) {
        // TODO - inmprove type safety of all 3 options
        setSelectedPlatformOptions(
          pull(valArray, "").join(",") as CommaSeparatedPlatformString
        );
      } else if (valArray.length > 1 && valArray.indexOf("") > -1) {
        setSelectedPlatformOptions("");
      } else {
        setSelectedPlatformOptions(values as CommaSeparatedPlatformString);
      }
    },
    [setSelectedPlatformOptions]
  );

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
          label="Frequency"
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
                      for this query until a frequency is set.
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
                  excludeTooltip
                />
              </b>
              .
            </>
          }
        />
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
              options={SCHEDULE_PLATFORM_DROPDOWN_OPTIONS}
              placeholder="Select"
              label="Platforms"
              onChange={onChangeSelectPlatformOptions}
              value={selectedPlatformOptions}
              multi
              wrapperClassName={`${baseClass}__form-field form-field--platform`}
              helpText="By default, your query collects data on all compatible platforms."
            />
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
            variant="brand"
            className="save-query-loading"
            isLoading={isLoading}
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
