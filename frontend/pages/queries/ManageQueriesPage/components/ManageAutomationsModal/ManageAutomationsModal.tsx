import React, { useState, useEffect } from "react";
import { omit } from "lodash";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
import CustomLink from "components/CustomLink/CustomLink";
import Checkbox from "components/forms/fields/Checkbox/Checkbox";
import TooltipWrapper from "components/TooltipWrapper/TooltipWrapper";
import Icon from "components/Icon/Icon";

import { ISchedulableQuery } from "interfaces/schedulable_query";

interface IFrequencyIndicator {
  frequency: number;
  checked: boolean;
}
interface IManageAutomationsModalProps {
  isUpdatingAutomations: boolean;
  handleSubmit: (formData: any) => void; // TODO
  onCancel: () => void;
  togglePreviewDataModal: () => void;
  availableQueries?: ISchedulableQuery[];
  automatedQueryIds: number[];
  logDestination: string;
}

interface ICheckedQuery {
  name?: string;
  id: number;
  isChecked: boolean;
  interval: number;
}

const useCheckboxListStateManagement = (
  allQueries: ISchedulableQuery[],
  automatedQueryIds: number[] | undefined
) => {
  const [queryItems, setQueryItems] = useState<ICheckedQuery[]>(() => {
    return allQueries.map(({ name, id, interval }) => ({
      name,
      id,
      isChecked: !!automatedQueryIds?.includes(id),
      interval,
    }));
  });

  const updateQueryItems = (queryId: number) => {
    setQueryItems((prevItems) =>
      prevItems.map((query) =>
        query.id !== queryId ? query : { ...query, isChecked: !query.isChecked }
      )
    );
  };

  return { queryItems, updateQueryItems };
};

const baseClass = "manage-automations-modal";

const ManageAutomationsModal = ({
  isUpdatingAutomations,
  automatedQueryIds,
  handleSubmit,
  onCancel,
  togglePreviewDataModal,
  availableQueries,
  logDestination,
}: IManageAutomationsModalProps): JSX.Element => {
  // TODO: Error handling, if any
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  console.log("MANAGEAUTOMATIONSMODAL automatedQueryIds", automatedQueryIds);
  const { queryItems, updateQueryItems } = useCheckboxListStateManagement(
    availableQueries || [],
    automatedQueryIds || []
  );

  const onSubmit = (evt: React.MouseEvent<HTMLFormElement> | KeyboardEvent) => {
    evt.preventDefault();

    console.log("automatedQueryIds", automatedQueryIds);
    const newQueryIds: number[] = [];
    queryItems?.forEach((p) => p.isChecked && newQueryIds.push(p.id));

    console.log("newQueryIds", newQueryIds);
    handleSubmit(newQueryIds);
  };

  useEffect(() => {
    const listener = (event: KeyboardEvent) => {
      if (event.code === "Enter" || event.code === "NumpadEnter") {
        event.preventDefault();
        onSubmit(event);
      }
    };
    document.addEventListener("keydown", listener);
    return () => {
      document.removeEventListener("keydown", listener);
    };
  });

  const renderLogDestination = () => {
    const readableLogDestination = () => {
      switch (logDestination) {
        case "filesystem":
          return "Filesystem";
        case "firehose":
          return "Amazon Kinesis Data Firehose";
        case "kinesis":
          return "Amazon Kinesis Data Streams";
        case "lambda":
          return "AWS Lambda";
        case "pubsub":
          return "Google Cloud Pub/Sub";
        case "kafta":
          return "Apache Kafka";
        case "stdout":
          return "Standard output (stdout)";
        case "":
          return "Not configured";
        default:
          return logDestination;
      }
    };

    const tooltipText = () => {
      switch (logDestination) {
        case "filesystem":
          return `Each time a query runs, the data is sent to <br />
            /var/log/osquery/osqueryd.snapshots.log <br />
            in each host&apos;s filesystem.`;
        case "firehose":
          return `Each time a query runs, the data is sent to <br />
            Amazon Kinesis Data Firehose.`;
        case "kinesis":
          return `Each time a query runs, the data is sent to <br />
            Amazon Kinesis Data Streams.`;
        case "lambda":
          return `
            Each time a query runs, the data <br />is sent to AWS Lambda.
          `;
        case "pubsub":
          return `Each time a query runs, the data is <br />sent to Google Cloud Pub/Sub.`;
        case "kafta":
          return `Each time a query runs, the data <br />is sent to Apache Kafka.`;
        case "stdout":
          return `Each time a query runs, the data is sent to <br />
            standard output (stdout) on the Fleet server.`;
        case "":
          return "Please configure a log destination.";
        default:
          return "No additional information is available about this log destination.";
      }
    };

    return (
      <TooltipWrapper tipContent={tooltipText()}>
        {readableLogDestination()}
      </TooltipWrapper>
    );
  };

  const renderScheduleIndicator = ({
    frequency,
    checked,
  }: IFrequencyIndicator) => {
    const readableQueryFrequency = () => {
      switch (frequency) {
        case 0:
          return "Never";
        case 300:
        case 600:
        case 900:
        case 1800: // 5, 10, 15, 30 minutes
          return `${(frequency / 60).toString()} minutes`;
        case 3600:
          return "Hourly";
        case 21600:
        case 43200: // 6, 12 hours
          return `${(frequency / 3600).toString()} hours`;
        case 86400:
          return "Daily";
        case 604800:
          return "Weekly";
        default:
          return "Unknown";
      }
    };

    const frequencyIcon = () => {
      switch (frequency) {
        case 0:
          return checked ? (
            <Icon size="small" name="warning" />
          ) : (
            <Icon size="small" name="clock" color="ui-fleet-black-33" />
          );
        default:
          return <Icon size="small" name="clock" />;
      }
    };
    return (
      <div
        className={`${baseClass}__schedule-indicator
        ${frequency === 0 && !checked && "grey"}`}
      >
        {frequencyIcon()}
        {readableQueryFrequency()}
      </div>
    );
  };

  return (
    <Modal
      title={"Manage automations"}
      onExit={onCancel}
      className={baseClass}
      width="large"
    >
      <div className={baseClass}>
        <div className={`${baseClass}__heading`}>
          Query automations let you send data to your log destination on a
          schedule. Data is sent according to a query’s frequency.
        </div>
        {availableQueries?.length ? (
          <div className={`${baseClass}__select`}>
            <p>
              <strong>Choose which queries will send data:</strong>
            </p>
            <div className={`${baseClass}__checkboxes`}>
              {queryItems &&
                queryItems.map((queryItem) => {
                  const { isChecked, name, id, interval } = queryItem;
                  return (
                    <div key={id} className={`${baseClass}__query-item`}>
                      <Checkbox
                        value={isChecked}
                        name={name}
                        onChange={() => {
                          updateQueryItems(id);
                          !isChecked &&
                            setErrors((errs) => omit(errs, "queryItems"));
                        }}
                      >
                        {name}
                      </Checkbox>
                      {renderScheduleIndicator({
                        frequency: interval,
                        checked: isChecked,
                      })}
                    </div>
                  );
                })}
            </div>
          </div>
        ) : (
          <div className={`${baseClass}__no-queries`}>
            <b>You have no queries.</b>
            <p>Add a query to turn on automations.</p>
          </div>
        )}
        <div className={`${baseClass}__log-destination`}>
          <p>
            <strong>Log destination:</strong>
          </p>
          <div className={`${baseClass}__selection`}>
            {renderLogDestination()}
          </div>
          <div className={`${baseClass}__configure`}>
            Users with the admin role can&nbsp;
            <CustomLink
              url="https://fleetdm.com/docs/using-fleet/log-destinations"
              text="configure a different log destination"
              newTab
            />
          </div>
        </div>
        <InfoBanner className={`${baseClass}__supported-platforms`}>
          Automations currently run on macOS, Windows, and Linux hosts.
          <br />
          Interested in query automations for your Chromebooks? &nbsp;
          <CustomLink
            url="https://fleetdm.com/contact"
            text="Let us know"
            newTab
          />
        </InfoBanner>
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
          <div className="modal-cta-wrap">
            <Button
              type="submit"
              variant="brand"
              onClick={onSubmit}
              className="save-loading"
              isLoading={isUpdatingAutomations}
            >
              Save
            </Button>
            <Button onClick={onCancel} variant="inverse">
              Cancel
            </Button>
          </div>
        </div>
      </div>
    </Modal>
  );
};

export default ManageAutomationsModal;
