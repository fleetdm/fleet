import React, { useState, useEffect } from "react";
import { omit } from "lodash";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
import CustomLink from "components/CustomLink/CustomLink";
import Checkbox from "components/forms/fields/Checkbox/Checkbox";
import TooltipWrapper from "components/TooltipWrapper/TooltipWrapper";

import { ISchedulableQuery } from "interfaces/schedulable_query";

// TODO
interface ISchedulableQueriesScheduled {
  query_ids: number[];
}
interface IManageAutomationsModalProps {
  isUpdatingAutomations: boolean;
  handleSubmit: (formData: any) => void; // TODO
  onCancel: () => void;
  togglePreviewDataModal: () => void;
  availableQueries?: ISchedulableQuery[];
  scheduledQueriesConfig: ISchedulableQueriesScheduled;
}

interface ICheckedPolicy {
  name?: string;
  id: number;
  isChecked: boolean;
}

const useCheckboxListStateManagement = (
  allQueries: ISchedulableQuery[],
  automatedQueries: number[] | undefined
) => {
  const [queryItems, setQueryItems] = useState<ICheckedPolicy[]>(() => {
    return allQueries.map(({ name, id }) => ({
      name,
      id,
      isChecked: !!automatedQueries?.includes(id),
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
  scheduledQueriesConfig,
  handleSubmit,
  onCancel,
  togglePreviewDataModal,
  availableQueries,
}: IManageAutomationsModalProps): JSX.Element => {
  // TODO: Error handling, if any
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  const { queryItems, updateQueryItems } = useCheckboxListStateManagement(
    availableQueries || [],
    scheduledQueriesConfig?.query_ids || []
  );

  const onSubmit = (evt: React.MouseEvent<HTMLFormElement> | KeyboardEvent) => {
    evt.preventDefault();

    const newQueryIds: number[] = [];
    queryItems?.forEach((p) => p.isChecked && newQueryIds.push(p.id));

    const newErrors = { ...errors };

    handleSubmit(newQueryIds);

    setErrors(newErrors);
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
    // TODO: tip content
    // TODO: dynamic log destination
    return (
      <TooltipWrapper
        tipContent={`Each time a query runs, the data is sent to /var/log/osquery/osqueryd.snapshots.log in each host’s filesystem.`}
      >
        Filesystem
      </TooltipWrapper>
    );
  };
  return (
    <Modal title={"Manage automations"} onExit={onCancel} className={baseClass}>
      <div className={baseClass}>
        <p className={`${baseClass}__heading`}>
          Query automations let you send data to your log destination on a
          schedule. Data is sent according to a query’s frequency.
        </p>
        <div className={`${baseClass}__select`}>
          {availableQueries?.length ? (
            <div className={`${baseClass}__query-select-items`}>
              <p>
                <strong>Choose which queries will send data:</strong>
              </p>
              <div className={`${baseClass}__checkboxes`}>
                {queryItems &&
                  queryItems.map((queryItem) => {
                    const { isChecked, name, id } = queryItem;
                    return (
                      <div key={id} className={`${baseClass}__team-item`}>
                        <Checkbox
                          value={isChecked}
                          name={name}
                          onChange={() => {
                            updateQueryItems(queryItem.id);
                            !isChecked &&
                              setErrors((errs) => omit(errs, "queryItems"));
                          }}
                        >
                          {name}
                        </Checkbox>
                      </div>
                    );
                  })}
              </div>
            </div>
          ) : (
            <div className={`${baseClass}__no-policies`}>
              <b>You have no policies.</b>
              <p>Add a policy to turn on automations.</p>
            </div>
          )}
          <div className={`${baseClass}__log-destination`}>
            <p>
              <strong>Log destination:</strong>
            </p>
            <div className={`${baseClass}__selection`}>
              {renderLogDestination()}
            </div>{" "}
            <div className={`${baseClass}__configure`}>
              Users with the admin role can&nbsp;
              <CustomLink
                url="https://fleetdm.com/docs/using-fleet/log-destinations"
                text="configure a different log destination"
                newTab
              />
            </div>
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
