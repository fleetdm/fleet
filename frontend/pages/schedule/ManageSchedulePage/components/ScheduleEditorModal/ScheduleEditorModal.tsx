/* This component is used for creating and editing both global and team scheduled queries */

import React, { useState, useCallback, useEffect } from "react";
// @ts-ignore
import Fleet from "fleet";
import { pull } from "lodash";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { IQuery } from "interfaces/query";
import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";
import { ITeamScheduledQuery } from "interfaces/team_scheduled_query";
import {
  FREQUENCY_DROPDOWN_OPTIONS,
  PLATFORM_DROPDOWN_OPTIONS,
  LOGGING_TYPE_OPTIONS,
  MIN_OSQUERY_VERSION_OPTIONS,
} from "utilities/constants";

import PreviewDataModal from "../PreviewDataModal";

const baseClass = "schedule-editor-modal";

interface IFormData {
  interval: number;
  name?: string;
  shard: number;
  query?: string;
  query_id?: number;
  logging_type: string;
  platform: string;
  version: string;
  team_id?: number;
}

interface IScheduleEditorModalProps {
  allQueries: IQuery[];
  onCancel: () => void;
  onScheduleSubmit: (
    formData: IFormData,
    editQuery: IGlobalScheduledQuery | ITeamScheduledQuery | undefined
  ) => void;
  editQuery?: IGlobalScheduledQuery | ITeamScheduledQuery;
  teamId?: number;
  togglePreviewDataModal: () => void;
  showPreviewDataModal: boolean;
}
interface INoQueryOption {
  id: number;
  name: string;
}

const generateLoggingType = (query: IGlobalScheduledQuery) => {
  if (query.snapshot) {
    return "snapshot";
  }
  if (query.removed) {
    return "differential";
  }
  return "differential_ignore_removals";
};

const generateLoggingDestination = (loggingConfig: string): string => {
  switch (loggingConfig) {
    case "filesystem":
      return "the filesystem";
    case "firehose":
      return "AWS Kinesis Firehose";
    case "kinesis":
      return "AWS Kinesis";
    case "lambda":
      return "AWS Lambda";
    case "pubsub":
      return "GCP PubSub";
    case "stdout":
      return "the standard output stream";
    default:
      return loggingConfig;
  }
};

const ScheduleEditorModal = ({
  onCancel,
  onScheduleSubmit,
  allQueries,
  editQuery,
  teamId,
  togglePreviewDataModal,
  showPreviewDataModal,
}: IScheduleEditorModalProps): JSX.Element => {
  const [loggingConfig, setLoggingConfig] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingError, setIsLoadingError] = useState(false);

  useEffect((): void => {
    const getConfigDestination = async (): Promise<void> => {
      try {
        const responseConfig = await Fleet.config.loadAll();
        setIsLoading(false);
        setLoggingConfig(responseConfig.logging.result.plugin);
      } catch (err) {
        setIsLoadingError(true);
        setIsLoading(false);
      }
    };
    getConfigDestination();
  }, []);

  const [showAdvancedOptions, setShowAdvancedOptions] = useState<boolean>(
    false
  );
  const [selectedQuery, setSelectedQuery] = useState<
    IGlobalScheduledQuery | INoQueryOption
  >();
  const [selectedFrequency, setSelectedFrequency] = useState<number>(
    editQuery ? editQuery.interval : 86400
  );
  const [
    selectedPlatformOptions,
    setSelectedPlatformOptions,
  ] = useState<string>(editQuery?.platform || "");
  const [selectedLoggingType, setSelectedLoggingType] = useState<string>(
    editQuery ? generateLoggingType(editQuery) : "snapshot"
  );
  const [
    selectedMinOsqueryVersionOptions,
    setSelectedMinOsqueryVersionOptions,
  ] = useState<string>(editQuery?.version || "");
  const [selectedShard, setSelectedShard] = useState<string>(
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

  const toggleAdvancedOptions = () => {
    setShowAdvancedOptions(!showAdvancedOptions);
  };

  const onChangeSelectQuery = useCallback(
    (queryId: string) => {
      const queryWithId: IQuery | undefined = allQueries.find(
        (query: IQuery) => query.id === parseInt(queryId, 10)
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
    (values: string) => {
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
    (value: string) => {
      setSelectedMinOsqueryVersionOptions(value);
    },
    [setSelectedMinOsqueryVersionOptions]
  );

  const onChangeShard = useCallback(
    (value: string) => {
      setSelectedShard(value);
    },
    [setSelectedShard]
  );

  const onFormSubmit = () => {
    const query_id = () => {
      if (editQuery) {
        return editQuery.query_id;
      }
      return selectedQuery?.id;
    };

    const name = () => {
      if (editQuery) {
        return editQuery.name;
      }
      return selectedQuery?.name;
    };

    onScheduleSubmit(
      {
        shard: parseInt(selectedShard, 10),
        interval: selectedFrequency,
        query_id: query_id(),
        name: name(),
        logging_type: selectedLoggingType,
        platform: selectedPlatformOptions,
        version: selectedMinOsqueryVersionOptions,
        team_id: teamId,
      },
      editQuery
    );
  };

  if (showPreviewDataModal) {
    return <PreviewDataModal onCancel={togglePreviewDataModal} />;
  }

  return (
    <Modal
      title={editQuery?.name || "Schedule editor"}
      onExit={onCancel}
      className={baseClass}
    >
      <form className={`${baseClass}__form`}>
        {!editQuery && (
          <Dropdown
            searchable
            options={createQueryDropdownOptions()}
            onChange={onChangeSelectQuery}
            placeholder={"Select query"}
            value={selectedQuery?.id}
            wrapperClassName={`${baseClass}__select-query-dropdown-wrapper`}
          />
        )}
        <Dropdown
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
            Your configured log destination is <b>{loggingConfig}</b>.
          </p>
          <p>
            This means that when this query is run on your hosts, the data will
            be sent to {generateLoggingDestination(loggingConfig)}.
          </p>
          <p>
            Check out the Fleet documentation on&nbsp;
            <a
              href="https://github.com/fleetdm/fleet/blob/6649d08a05799811f6fb0566947946edbfebf63e/docs/2-Deploying/2-Configuration.md#osquery_result_log_plugin"
              target="_blank"
              rel="noopener noreferrer"
            >
              how to configure a different log destination
            </a>
            .
          </p>
        </InfoBanner>
        <div>
          <Button
            variant="unstyled"
            className={`${showAdvancedOptions ? "upcarat" : "downcarat"} 
               ${baseClass}__advanced-options-button`}
            onClick={toggleAdvancedOptions}
          >
            {showAdvancedOptions
              ? "Hide advanced options"
              : "Show advanced options"}
          </Button>
          {showAdvancedOptions && (
            <div>
              <Dropdown
                options={LOGGING_TYPE_OPTIONS}
                onChange={onChangeSelectLoggingType}
                placeholder="Select"
                value={selectedLoggingType}
                label="Logging"
                wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--logging`}
              />
              <Dropdown
                options={PLATFORM_DROPDOWN_OPTIONS}
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
                label="Shard"
                type="number"
              />
            </div>
          )}
        </div>
        <div className={`${baseClass}__btn-wrap`}>
          <div className={`${baseClass}__preview-btn-wrap`}>
            <Button
              type="button"
              variant="inverse"
              onClick={togglePreviewDataModal}
            >
              Preview data
            </Button>
          </div>
          <div className={`${baseClass}__cta-btn-wrap`}>
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
              disabled={!selectedQuery && !editQuery}
            >
              Schedule
            </Button>
          </div>
        </div>
      </form>
    </Modal>
  );
};

export default ScheduleEditorModal;
