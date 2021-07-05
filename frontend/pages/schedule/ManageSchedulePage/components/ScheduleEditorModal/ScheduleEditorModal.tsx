import React, { useState, useCallback } from "react";

// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { IQuery } from "interfaces/query";

const baseClass = "schedule-editor-modal";

interface IScheduleEditorModalProps {
  allQueries: IQuery[];
  onCancel: () => void;
  onScheduleSubmit: (formData: any) => void;
  defaultLoggingType: string | null;
  validationErrors?: any[]; // TODO: proper interface for validationErrors
}
interface IFrequencyOption {
  value: number;
  label: string;
}

const ScheduleEditorModal = (props: IScheduleEditorModalProps): JSX.Element => {
  const { onCancel, onScheduleSubmit, allQueries, defaultLoggingType } = props;

  // 7/5 TODO: Render selected query on dropdown
  // Need: How the return value object changes the selection
  // Need: How does {...fields.____} work with our codebase
  // Query dropdown
  interface INoQueryOption {
    id: number;
    name: string;
  }
  const [selectedQuery, setSelectedQuery] = useState<IQuery | INoQueryOption>();

  const createQueryDropdownOptions = () => {
    const queryOptions = allQueries.map((q: any) => {
      return {
        value: String(q.id),
        label: q.name,
      };
    });
    return [...queryOptions];
  };

  const onChangeSelectQuery = useCallback(
    (queryId: number | string) => {
      const queryWithId = allQueries.find(
        (query: IQuery) => query.id === queryId
      );
      setSelectedQuery(queryWithId as IQuery);
    },
    [allQueries, setSelectedQuery]
  );
  // End query dropdown

  // Frequency dropdown
  const frequencyDropdownOptions = [
    { value: 3600, label: "Every hour" },
    { value: 21600, label: "Every 6 hours" },
    { value: 43200, label: "Every 12 hours" },
    { value: 86400, label: "Every day" },
    { value: 604800, label: "Every week" },
  ];

  const [selectedFrequency, setSelectedFrequency] = useState(86400);

  const onChangeSelectFrequency = useCallback(
    (value: number) => {
      setSelectedFrequency(value);
    },
    [setSelectedFrequency]
  );
  // End Frequency dropdown

  // Advanced Options
  const [showAdvancedOptions, setShowAdvancedOptions] = useState(false);

  const toggleAdvancedOptions = () => {
    setShowAdvancedOptions(!showAdvancedOptions);
  };

  const fieldNames = [
    "query_id",
    "interval",
    "logging_type",
    "platform",
    "shard",
    "version",
  ];

  const platformOptions = [
    { label: "All", value: "" },
    { label: "Windows", value: "windows" },
    { label: "Linux", value: "linux" },
    { label: "macOS", value: "darwin" },
  ];

  const loggingTypeOptions = [
    { label: "Differential", value: "differential" },
    {
      label: "Differential (Ignore Removals)",
      value: "differential_ignore_removals",
    },
    { label: "Snapshot", value: "snapshot" },
  ];

  const [selectedLoggingType, setSelectedLoggingType] = useState("snapshot");

  const onChangeSelectLoggingType = useCallback(
    (value: string) => {
      setSelectedLoggingType(value);
    },
    [setSelectedLoggingType]
  );
  const minOsqueryVersionOptions = [
    { label: "All", value: "" },
    { label: "4.7.0 +", value: "4.7.0" },
    { label: "4.6.0 +", value: "4.6.0" },
    { label: "4.5.1 +", value: "4.5.1" },
    { label: "4.5.0 +", value: "4.5.0" },
    { label: "4.4.0 +", value: "4.4.0" },
    { label: "4.3.0 +", value: "4.3.0" },
    { label: "4.2.0 +", value: "4.2.0" },
    { label: "4.1.2 +", value: "4.1.2" },
    { label: "4.1.1 +", value: "4.1.1" },
    { label: "4.1.0 +", value: "4.1.0" },
    { label: "4.0.2 +", value: "4.0.2" },
    { label: "4.0.1 +", value: "4.0.1" },
    { label: "4.0.0 +", value: "4.0.0" },
    { label: "3.4.0 +", value: "3.4.0" },
    { label: "3.3.2 +", value: "3.3.2" },
    { label: "3.3.1 +", value: "3.3.1" },
    { label: "3.2.6 +", value: "3.2.6" },
    { label: "2.2.1 +", value: "2.2.1" },
    { label: "2.2.0 +", value: "2.2.0" },
    { label: "2.1.2 +", value: "2.1.2" },
    { label: "2.1.1 +", value: "2.1.1" },
    { label: "2.0.0 +", value: "2.0.0" },
    { label: "1.8.2 +", value: "1.8.2" },
    { label: "1.8.1 +", value: "1.8.1" },
  ];

  const [
    selectedMinOsqueryVersionOptions,
    setSelectedMinOsqueryVersionOptions,
  ] = useState(null);

  const onChangeMinOsqueryVersionOptions = useCallback(
    (value: any) => {
      setSelectedMinOsqueryVersionOptions(value);
    },
    [setSelectedMinOsqueryVersionOptions]
  );

  const [selectedShard, setSelectedShard] = useState(null);

  const onChangeShard = useCallback(
    (value: any) => {
      setSelectedShard(value);
    },
    [setSelectedShard]
  );

  // 7/5 TODO: How to create this platform chooser in functional/typescript
  // This is written in class/javascript on ConfigurePackQueryForm.jsx Line 95
  const handlePlatformChoice = (value: string) => {
    //   const {
    //     fields: { platform },
    //   } = this.props;
    //   const valArray = value.split(",");
    //   // Remove All if another OS is chosen
    //   if (valArray.indexOf("") === 0 && valArray.length > 1) {
    //     return platform.onChange(pull(valArray, "").join(","));
    //   }
    //   // Remove OS if All is chosen
    //   if (valArray.length > 1 && valArray.indexOf("") > -1) {
    //     return platform.onChange("");
    //   }
    //   return platform.onChange(value);
  };

  // End Advanced Options

  return (
    <Modal title={"Schedule editor"} onExit={onCancel} className={baseClass}>
      <form className={`${baseClass}__form`}>
        <Dropdown
          wrapperClassName={`${baseClass}__select-query-dropdown-wrapper`}
          value={selectedQuery && selectedQuery.id}
          options={createQueryDropdownOptions()}
          onChange={onChangeSelectQuery}
          placeholder={"Select query"}
          searchable={true}
        />
        <Dropdown
          // {...fields.frequency}
          searchable={false}
          options={frequencyDropdownOptions}
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
                options={loggingTypeOptions}
                onChange={onChangeSelectLoggingType}
                placeholder="Select"
                value={selectedLoggingType}
                label="Logging"
                wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--logging`}
              />
              <Dropdown
                // {...fields.platform}
                options={platformOptions}
                placeholder="Select"
                label="Platform"
                onChange={handlePlatformChoice}
                multi
                wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--platform`}
              />
              <Dropdown
                // {...fields.version}
                options={minOsqueryVersionOptions}
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
            onClick={onScheduleSubmit}
          >
            Schedule
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default ScheduleEditorModal;
