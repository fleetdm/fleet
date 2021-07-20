import React, { useState, useCallback } from "react";

// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore No longer using Form component 7/7
// import Form from "components/forms/Form";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";
import {
  FREQUENCY_DROPDOWN_OPTIONS,
  PLATFORM_OPTIONS,
  LOGGING_TYPE_OPTIONS,
  MIN_OSQUERY_VERSION_OPTIONS,
} from "utilities/constants";

// Are these needed? 7/8
// import endpoints from "fleet/endpoints";
// import AutocompleteDropdown from "pages/admin/TeamManagementPage/TeamDetailsWrapper/MembersPagePage/components/AutocompleteDropdown";
// import { IDropdownOption } from "interfaces/dropdownOption";

const baseClass = "schedule-editor-modal";
interface IScheduleEditorModalProps {
  allQueries: IGlobalScheduledQuery[];
  onCancel: () => void;
  onScheduleSubmit: (formData: any) => void;
  validationErrors?: any[]; // TODO: proper interface for validationErrors
}
interface IFrequencyOption {
  value: number;
  label: string;
}

interface INoQueryOption {
  id: number;
  name: string;
}

const ScheduleEditorModal = ({
  onCancel,
  onScheduleSubmit,
  allQueries,
}: IScheduleEditorModalProps): JSX.Element => {
  const [showAdvancedOptions, setShowAdvancedOptions] = useState<boolean>(
    false
  );
  const [selectedQuery, setSelectedQuery] = useState<
    IGlobalScheduledQuery | INoQueryOption
  >();
  const [selectedFrequency, setSelectedFrequency] = useState<number>(86400);
  const [
    selectedPlatformOptions,
    setSelectedPlatformOptions,
  ] = useState<string>("");
  const [selectedLoggingType, setSelectedLoggingType] = useState<string>(
    "snapshot"
  );
  const [
    selectedMinOsqueryVersionOptions,
    setSelectedMinOsqueryVersionOptions,
  ] = useState(null);
  const [selectedShard, setSelectedShard] = useState(null);

  console.log("selectedLoggingType scheduleEditorModal", selectedLoggingType);
  const createQueryDropdownOptions = () => {
    const queryOptions = allQueries.map((q: any) => {
      return {
        value: String(q.id),
        label: q.name,
      };
    });
    return [...queryOptions];
  };

  const toggleAdvancedOptions = () => {
    setShowAdvancedOptions(!showAdvancedOptions);
  };

  const onChangeSelectQuery = useCallback(
    (queryId: number | string) => {
      const queryWithId: IGlobalScheduledQuery | undefined = allQueries.find(
        (query: IGlobalScheduledQuery) => query.id == queryId
      );
      setSelectedQuery(queryWithId);
    },
    [allQueries, setSelectedQuery]
  );

  const onChangeSelectFrequency = useCallback(
    (value: number) => {
      setSelectedFrequency(value);
    },
    [setSelectedFrequency]
  );

  const onChangeSelectPlatformOptions = useCallback(
    (values) => {
      setSelectedPlatformOptions(values);
    },
    [setSelectedPlatformOptions]
  );

  const onChangeSelectLoggingType = useCallback(
    (value: string) => {
      setSelectedLoggingType(value);
    },
    [setSelectedLoggingType]
  );

  const onChangeMinOsqueryVersionOptions = useCallback(
    (value: any) => {
      setSelectedMinOsqueryVersionOptions(value);
    },
    [setSelectedMinOsqueryVersionOptions]
  );

  const onChangeShard = useCallback(
    (value: any) => {
      setSelectedShard(value);
    },
    [setSelectedShard]
  );

  // TODO: Validators
  // TODO: figure out differential/removal bug

  const onFormSubmit = () => {
    console.log(
      "\nTODO: Fix selectedLoggingType onFormSubmit,",
      selectedLoggingType
    );
    debugger;
    onScheduleSubmit({
      shard: selectedShard,
      interval: selectedFrequency,
      query_id: selectedQuery?.id,
      snapshot: selectedLoggingType === "snapshot",
      removed: selectedLoggingType === "differential",
      platform: selectedPlatformOptions,
      version: selectedMinOsqueryVersionOptions,
    });
  };

  return (
    <Modal title={"Schedule editor"} onExit={onCancel} className={baseClass}>
      <form className={`${baseClass}__form`}>
        <Dropdown
          // {...fields.query_id}
          searchable={true}
          options={createQueryDropdownOptions()}
          onChange={onChangeSelectQuery}
          placeholder={"Select query"}
          value={selectedQuery?.id}
          wrapperClassName={`${baseClass}__select-query-dropdown-wrapper`}
        />
        <Dropdown
          // {...fields.frequency}
          searchable={false}
          options={FREQUENCY_DROPDOWN_OPTIONS}
          onChange={onChangeSelectFrequency}
          placeholder={"Every day"}
          value={selectedFrequency}
          label={"Choose a frequency and then run this query on a schedule"}
          wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--frequency`}
        />
        <InfoBanner className={`${baseClass}__sandbox-info`}>
          <p>
            Your configured log destination is <b>filesystem</b>.
          </p>
          <p>
            This means that when this query is run on your hosts, the data will
            be sent to the filesystem.
          </p>
          <p>
            Check out the Fleet documentation on&nbsp;
            <a
              href="https://github.com/fleetdm/fleet/blob/6649d08a05799811f6fb0566947946edbfebf63e/docs/2-Deploying/2-Configuration.md#osquery_result_log_plugin"
              target="_blank"
              rel="noopener noreferrer"
            >
              how configure a different log destination.&nbsp;
              <FleetIcon name="external-link" />
            </a>
          </p>
        </InfoBanner>
        <div>
          <Button
            variant="unstyled"
            className={
              (showAdvancedOptions ? "upcarat" : "downcarat") +
              ` ${baseClass}__advanced-options-button`
            }
            onClick={toggleAdvancedOptions}
          >
            {showAdvancedOptions
              ? "Hide advanced options"
              : "Show advanced options"}
          </Button>
          {showAdvancedOptions && (
            <div>
              <Dropdown
                // {...fields.logging_type}
                options={LOGGING_TYPE_OPTIONS}
                onChange={onChangeSelectLoggingType}
                placeholder="Select"
                value={selectedLoggingType}
                label="Logging"
                wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--logging`}
              />
              <Dropdown
                // {...fields.platform}
                options={PLATFORM_OPTIONS}
                placeholder="Select"
                label="Platform"
                onChange={onChangeSelectPlatformOptions}
                value={selectedPlatformOptions}
                multi
                wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--platform`}
              />
              <Dropdown
                // {...fields.version}
                options={MIN_OSQUERY_VERSION_OPTIONS}
                onChange={onChangeMinOsqueryVersionOptions}
                placeholder="Select"
                value={selectedMinOsqueryVersionOptions}
                label="Minimum osquery version"
                wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--osquer-vers`}
              />
              <InputField
                // {...fields.shard}
                onChange={onChangeShard}
                inputWrapperClass={`${baseClass}__form-field ${baseClass}__form-field--shard`}
                value={selectedShard}
                placeholder="- - -"
                label="Shard"
                type="number"
              />
            </div>
          )}
        </div>

        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="inverse"
          >
            Cancel
          </Button>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="brand"
            onClick={onFormSubmit}
          >
            Schedule
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default ScheduleEditorModal;
