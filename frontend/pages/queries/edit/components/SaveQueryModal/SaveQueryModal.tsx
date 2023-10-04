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
import TooltipWrapper from "components/TooltipWrapper";
import { Link } from "react-router";
import Icon from "components/Icon";
import { IConfig } from "interfaces/config";
import InfoBanner from "components/InfoBanner";

const baseClass = "save-query-modal";
export interface ISaveQueryModalProps {
  queryValue: string;
  apiTeamIdForQuery?: number; // query will be global if omitted
  isLoading: boolean;
  saveQuery: (formData: ICreateQueryRequestBody) => void;
  toggleSaveQueryModal: () => void;
  backendValidators: { [key: string]: string };
  existingQuery?: ISchedulableQuery;
  appConfig?: IConfig;
  isLoadingAppConfig?: boolean;
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
  appConfig,
  isLoadingAppConfig,
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
  const [discardData, setDiscardData] = useState(false);
  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);
  const [forceEditDiscardData, setForceEditDiscardData] = useState(false);

  const toggleAdvancedOptions = () => {
    setShowAdvancedOptions(!showAdvancedOptions);
  };

  const query_reports_disabled =
    appConfig?.server_settings?.query_reports_disabled;

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

  const renderDiscardDataOption = () => {
    const disable = query_reports_disabled && !forceEditDiscardData;
    return (
      <>
        {["differential", "differential_ignore_removals"].includes(
          selectedLoggingType
        ) && (
          <InfoBanner color="purple-bold-border">
            <>
              The <b>Discard data</b> setting is ignored when differential
              logging is enabled. This <br />
              query&apos;s results will not be saved in Fleet.
            </>
          </InfoBanner>
        )}
        <Checkbox
          name="discardData"
          onChange={setDiscardData}
          value={discardData}
          wrapperClassName={
            disable ? `${baseClass}__disabled-discard-data-checkbox` : ""
          }
        >
          Discard data
        </Checkbox>
        <div className="help-text">
          {disable ? (
            <>
              This setting is ignored because query reports in Fleet have been{" "}
              <TooltipWrapper
                // TODO - use JSX once new tooltipwrapper is merged
                tipContent={
                  "A Fleet administrator can enable query reports under <br />\
                  <b>Organization settings > Advanced options > Disable  query reports</b>."
                }
                position="bottom"
              >
                <>globally disabled.</>
              </TooltipWrapper>{" "}
              <Link
                to={""}
                onClick={() => {
                  setForceEditDiscardData(true);
                }}
                className={`${baseClass}__edit-anyway`}
              >
                <>
                  Edit anyway
                  <Icon
                    name="chevron"
                    direction="right"
                    color="core-fleet-blue"
                    size="small"
                  />
                </>
              </Link>
            </>
          ) : (
            <>
              The most recent results for each host will not be available in
              Fleet.
              <br />
              Data will still be sent to your log destination if{" "}
              <b>automations</b> are <b>on</b>.
            </>
          )}
        </div>
      </>
    );
  };
  return (
    <Modal title={"Save query"} onExit={toggleSaveQueryModal}>
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
          ignore1password
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
        <div className="help-text">
          This is how often your query collects data.
        </div>
        <Checkbox
          name="observerCanRun"
          onChange={setObserverCanRun}
          value={observerCanRun}
          wrapperClassName={`${baseClass}__observer-can-run-wrapper`}
        >
          Observers can run
        </Checkbox>
        <div className="help-text">
          Users with the Observer role will be able to run this query as a live
          query.
        </div>
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
            <div className="help-text">
              By default, your query collects data on all compatible platforms.
            </div>
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
            {!isLoadingAppConfig && renderDiscardDataOption()}
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
