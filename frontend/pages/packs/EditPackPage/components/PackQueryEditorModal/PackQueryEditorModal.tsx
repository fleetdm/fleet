/* This component is used for creating and editing pack queries */

import React, { useState } from "react";
import { pull } from "lodash";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { IQuery } from "interfaces/query";
import { IScheduledQuery } from "interfaces/scheduled_query";
import {
  SCHEDULE_PLATFORM_DROPDOWN_OPTIONS,
  LOGGING_TYPE_OPTIONS,
  MAX_OSQUERY_SCHEDULED_QUERY_INTERVAL,
  MIN_OSQUERY_VERSION_OPTIONS,
} from "utilities/constants";

const baseClass = "pack-query-editor-modal";

interface IFormData {
  interval: number;
  name?: string;
  shard: number;
  query?: string;
  query_id?: number;
  snapshot: boolean;
  removed: boolean;
  platform: string;
  version: string;
  pack_id: number;
}

interface IPackQueryEditorModalProps {
  allQueries: IQuery[];
  onCancel: () => void;
  onPackQueryFormSubmit: (
    formData: IFormData,
    editQuery: IScheduledQuery | undefined
  ) => void;
  editQuery?: IScheduledQuery;
  packId: number;
  isUpdatingPack: boolean;
}
interface INoQueryOption {
  id: number;
  name: string;
}

const generateLoggingType = (query: IScheduledQuery) => {
  if (query.snapshot) {
    return "snapshot";
  }
  if (query.removed) {
    return "differential";
  }
  return "differential_ignore_removals";
};

const PackQueryEditorModal = ({
  onCancel,
  onPackQueryFormSubmit,
  allQueries,
  editQuery,
  packId,
  isUpdatingPack,
}: IPackQueryEditorModalProps): JSX.Element => {
  const [selectedQuery, setSelectedQuery] = useState<
    IScheduledQuery | INoQueryOption
  >();
  const [selectedFrequency, setSelectedFrequency] = useState(
    editQuery?.interval.toString() || ""
  );
  const [errorFrequency, setErrorFrequency] = useState("");
  const [selectedPlatformOptions, setSelectedPlatformOptions] = useState(
    editQuery?.platform || ""
  );
  const [selectedLoggingType, setSelectedLoggingType] = useState(
    editQuery ? generateLoggingType(editQuery) : "snapshot"
  );
  const [selectedSnapshot, setSelectedSnapshot] = useState(
    selectedLoggingType === "snapshot"
  );
  const [selectedRemoved, setSelectedRemoved] = useState(
    selectedLoggingType === "differential"
  );
  const [
    selectedMinOsqueryVersionOptions,
    setSelectedMinOsqueryVersionOptions,
  ] = useState(editQuery?.version || "");
  const [selectedShard, setSelectedShard] = useState(
    editQuery?.shard ? editQuery?.shard.toString() : ""
  );

  const createQueryDropdownOptions = () => {
    const queryOptions = allQueries.map((q) => {
      return {
        value: String(q.id),
        label: q.name,
      };
    });
    return queryOptions;
  };

  const onChangeSelectQuery = (queryId: string) => {
    const queryWithId: IQuery | undefined = allQueries.find(
      (query: IQuery) => query.id === parseInt(queryId, 10)
    );
    setSelectedQuery(queryWithId);
  };

  const onChangeFrequency = (value: string) => {
    if (errorFrequency) {
      setErrorFrequency("");
    }
    setSelectedFrequency(value);
  };

  const onChangeSelectPlatformOptions = (values: string) => {
    const valArray = values.split(",");

    // Remove All if another OS is chosen
    // else if Remove OS if All is chosen
    if (valArray.indexOf("") === 0 && valArray.length > 1) {
      setSelectedPlatformOptions(pull(valArray, "").join(","));
    } else if (valArray.length > 1 && valArray.indexOf("") > -1) {
      setSelectedPlatformOptions("");
    } else {
      setSelectedPlatformOptions(values);
    }
  };

  const onChangeSelectLoggingType = (value: string) => {
    setSelectedLoggingType(value);
    setSelectedRemoved(value === "differential");
    setSelectedSnapshot(value === "snapshot");
  };

  const onChangeMinOsqueryVersionOptions = (value: string) => {
    setSelectedMinOsqueryVersionOptions(value);
  };

  const onChangeShard = (value: string) => {
    setSelectedShard(value);
  };

  const onFormSubmit = (): void => {
    setErrorFrequency("");
    const query_id = () => {
      if (editQuery) {
        return editQuery.query_id;
      }
      return selectedQuery?.id;
    };

    const frequency = parseInt(selectedFrequency, 10);
    if (!frequency || frequency < 0) {
      setErrorFrequency("Frequency must be an integer greater than zero");
      return;
    }
    if (frequency > MAX_OSQUERY_SCHEDULED_QUERY_INTERVAL) {
      setErrorFrequency(
        "Frequency must be an integer that does not exceed 604,800 (i.e. 7 days)"
      );
      return;
    }

    onPackQueryFormSubmit(
      {
        interval: parseInt(selectedFrequency, 10),
        pack_id: packId,
        platform: selectedPlatformOptions,
        query_id: query_id(),
        // name: name(), // pretty sure unneeded
        removed: selectedRemoved,
        snapshot: selectedSnapshot,
        shard: parseInt(selectedShard, 10),
        version: selectedMinOsqueryVersionOptions,
      },
      editQuery
    );
  };

  return (
    <Modal
      title={editQuery?.name || "Add query"}
      onExit={onCancel}
      className={baseClass}
    >
      <form className={`${baseClass}__form`}>
        {!editQuery && (
          <Dropdown
            searchable
            options={createQueryDropdownOptions()}
            onChange={onChangeSelectQuery}
            placeholder="Select query"
            value={selectedQuery?.id}
            wrapperClassName={`${baseClass}__select-query-dropdown-wrapper`}
            autoFocus
          />
        )}
        <InputField
          onChange={onChangeFrequency}
          error={errorFrequency}
          inputWrapperClass={`${baseClass}__form-field ${baseClass}__form-field--frequency`}
          value={selectedFrequency}
          placeholder="- - -"
          label="Frequency (seconds)"
          type="number"
        />
        <Dropdown
          options={LOGGING_TYPE_OPTIONS}
          onChange={onChangeSelectLoggingType}
          placeholder="Select"
          value={selectedLoggingType}
          label="Logging"
          wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--logging`}
        />
        <Dropdown
          options={SCHEDULE_PLATFORM_DROPDOWN_OPTIONS}
          placeholder="Select"
          label="Platform"
          onChange={onChangeSelectPlatformOptions}
          value={selectedPlatformOptions}
          multi
          wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--platform`}
        />
        <Dropdown
          options={MIN_OSQUERY_VERSION_OPTIONS}
          onChange={onChangeMinOsqueryVersionOptions}
          placeholder="Select"
          value={selectedMinOsqueryVersionOptions}
          label="Minimum osquery version"
          wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--osquer-vers`}
        />
        <InputField
          onChange={onChangeShard}
          inputWrapperClass={`${baseClass}__form-field ${baseClass}__form-field--shard`}
          value={selectedShard}
          placeholder="- - -"
          label="Shard (percentage)"
          type="number"
        />

        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="brand"
            onClick={onFormSubmit}
            disabled={!selectedQuery && !editQuery}
            className={`${editQuery?.name ? "save" : "add-query"}-loading`}
            isLoading={isUpdatingPack}
          >
            {editQuery?.name ? "Save" : "Add query"}
          </Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default PackQueryEditorModal;
