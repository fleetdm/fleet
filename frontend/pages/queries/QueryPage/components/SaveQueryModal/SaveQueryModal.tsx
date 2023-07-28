import React, { useState, useEffect, useCallback } from "react";
import { pull, size } from "lodash";

import useDeepEffect from "hooks/useDeepEffect";

import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import {
  FREQUENCY_DROPDOWN_OPTIONS,
  LOGGING_TYPE_OPTIONS,
  MIN_OSQUERY_VERSION_OPTIONS,
  SCHEDULE_PLATFORM_DROPDOWN_OPTIONS,
} from "utilities/constants";
import RevealButton from "components/buttons/RevealButton";
import { SelectedPlatformString } from "interfaces/platform";
import {
  ICreateQueryRequestBody,
  ISchedulableQuery,
  QueryLoggingOption,
} from "interfaces/schedulable_query";

const baseClass = "save-query-modal";
export interface ISaveQueryModalProps {
  queryValue: string;
  apiTeamIdForQuery?: number; // query will be global if omitted
  isLoading: boolean;
  saveQuery: (formData: ICreateQueryRequestBody) => void;
  toggleSaveQueryModal: () => void;
  backendValidators: { [key: string]: string };
  existingQuery?: ISchedulableQuery;
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
}: ISaveQueryModalProps): JSX.Element => {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [selectedFrequency, setSelectedFrequency] = useState(
    existingQuery?.interval ?? 3600
  );
  const [
    selectedPlatformOptions,
    setSelectedPlatformOptions,
  ] = useState<SelectedPlatformString>(existingQuery?.platform ?? "");
  const [
    selectedMinOsqueryVersionOptions,
    setSelectedMinOsqueryVersionOptions,
  ] = useState(existingQuery?.min_osquery_version ?? "");
  const [
    selectedLoggingType,
    setSelectedLoggingType,
  ] = useState<QueryLoggingOption>(existingQuery?.logging ?? "snapshot");
  const [observerCanRun, setObserverCanRun] = useState(false);
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

    const { valid, errors: newErrors } = validateQueryName(name);
    setErrors({
      ...errors,
      ...newErrors,
    });

    if (valid) {
      saveQuery({
        // from modal fields
        name,
        description,
        interval: selectedFrequency,
        observer_can_run: observerCanRun,
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
          pull(valArray, "").join(",") as SelectedPlatformString
        );
      } else if (valArray.length > 1 && valArray.indexOf("") > -1) {
        setSelectedPlatformOptions("");
      } else {
        setSelectedPlatformOptions(values as SelectedPlatformString);
      }
    },
    [setSelectedPlatformOptions]
  );

  return (
    <Modal title={"Save query"} onExit={toggleSaveQueryModal}>
      <>
        <form
          onSubmit={onClickSaveQuery}
          className={baseClass}
          autoComplete="off"
        >
          <InputField
            name="name"
            onChange={(value: string) => setName(value)}
            value={name}
            error={errors.name}
            inputClassName={`${baseClass}__name`}
            label="Name"
            placeholder="What is your query called?"
            autofocus
          />
          <InputField
            name="description"
            onChange={(value: string) => setDescription(value)}
            value={description}
            inputClassName={`${baseClass}__description`}
            label="Description"
            type="textarea"
            placeholder="What information does your query reveal? (optional)"
          />
          <Dropdown
            searchable={false}
            options={FREQUENCY_DROPDOWN_OPTIONS}
            onChange={(value: number) => {
              setSelectedFrequency(value);
            }}
            placeholder={"Every hour"}
            value={selectedFrequency}
            label="Frequency"
            wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--frequency`}
          />
          <p className="help-text">
            If automations are on, this is how often your query collects data.
          </p>
          <Checkbox
            name="observerCanRun"
            onChange={setObserverCanRun}
            value={observerCanRun}
            wrapperClassName={`${baseClass}__observer-can-run-wrapper`}
          >
            Observers can run
          </Checkbox>
          <p className="help-text">
            Users with the Observer role will be able to run this query as a
            live query.
          </p>
          <RevealButton
            isShowing={showAdvancedOptions}
            className={`${baseClass}__advanced-options-toggle`}
            hideText={"Hide advanced options"}
            showText={"Show advanced options"}
            caretPosition={"after"}
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
                wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--platform`}
              />
              <p className="help-text">
                If automations are turned on, your query collects data on
                compatible platforms.
                <br />
                If you want more control, override platforms.
              </p>
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
      </>
    </Modal>
  );
};

export default SaveQueryModal;
