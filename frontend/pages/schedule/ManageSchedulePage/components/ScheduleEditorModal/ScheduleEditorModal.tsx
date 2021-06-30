import React, { useState, useCallback } from "react";
import { pull } from "lodash";

import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { IQuery } from "interfaces/query";

const baseClass = "schedule-editor-modal";

export interface IScheduleEditorFormData {
  name: string;
}

interface IScheduleEditorModalProps {
  queries: IQuery[];
  onCancel: () => void;
  onSubmit: (formData: IScheduleEditorFormData) => void;
}
interface IFrequencyOption {
  value: number;
  label: string;
}

const ScheduleEditorModal = (props: IScheduleEditorModalProps): JSX.Element => {
  const { onCancel, onSubmit, queries } = props;

  // FUNCTIONALITY LATER 6/30
  // const onFormSubmit = useCallback(
  //   (evt) => {
  //     evt.preventDefault();
  //     onSubmit({
  //       name,
  //     });
  //   },
  //   [onSubmit, name]
  // );

  // Query dropdown
  interface INoQueryOption {
    id: number;
    name: string;
  }
  const [selectedQuery, setSelectedQuery] = useState<IQuery | INoQueryOption>();

  // const createQueryDropdownOptions = () => {
  //   console.log(queries);
  //   debugger;
  //   const queryOptions = queries.map((q) => {
  //     return {
  //       value: q.id,
  //       label: q.name,
  //     };
  //   });
  //   return [...queryOptions];
  // };

  const onChangeSelectQuery = useCallback(
    (queryId: number | string) => {
      const queryWithId = queries.find((query) => query.id === queryId);
      setSelectedQuery(queryWithId as IQuery);
    },
    [queries, setSelectedQuery]
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
        {/* <Dropdown
          wrapperClassName={`${baseClass}__select-query-dropdown-wrapper`}
          value={selectedQuery && selectedQuery.id}
          options={createQueryDropdownOptions()}
          onChange={onChangeSelectQuery}
          placeholder={"Select query"}
          searchable={true}
        /> */}
        <Dropdown
          // {...fields.frequency}
          wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--frequency`}
          label={"Choose a frequency and then run this query on a schedule"}
          value={selectedFrequency}
          options={frequencyDropdownOptions}
          onChange={onChangeSelectFrequency}
          placeholder={"Every day"}
          searchable={false}
        />
        <InfoBanner className={`${baseClass}__sandbox-info`}>
          <p>Your configured log destination is filesystem.</p>
          <p>
            This means that when this query is run on your hosts, the data will
            be sent to the filesystem.
          </p>
          <p>
            Check out the Fleet documentation on how configure a different log
            destination.
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
                placeholder="- - -"
                label="Logging"
                wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--logging`}
              />
              <Dropdown
                // {...fields.platform}
                options={platformOptions}
                placeholder="- - -"
                label="Platform"
                onChange={handlePlatformChoice}
                multi
                wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--platform`}
              />
              <Dropdown
                // {...fields.version}
                options={minOsqueryVersionOptions}
                placeholder="- - -"
                label="Minimum osquery version"
                wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--osquer-vers`}
              />
              <InputField
                // {...fields.shard}
                inputWrapperClass={`${baseClass}__form-field ${baseClass}__form-field--shard`}
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
            // onClick={onFormSubmit}
          >
            Schedule
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default ScheduleEditorModal;
