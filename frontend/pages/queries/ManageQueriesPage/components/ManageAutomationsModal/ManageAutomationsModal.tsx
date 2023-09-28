import React, { useState, useEffect } from "react";
import { omit } from "lodash";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
import CustomLink from "components/CustomLink/CustomLink";
import Checkbox from "components/forms/fields/Checkbox/Checkbox";
import QueryFrequencyIndicator from "components/QueryFrequencyIndicator/QueryFrequencyIndicator";
import LogDestinationIndicator from "components/LogDestinationIndicator/LogDestinationIndicator";

import { ISchedulableQuery } from "interfaces/schedulable_query";

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

  // Client side sort queries alphabetically
  const sortedAvailableQueries =
    availableQueries?.sort((a, b) =>
      a.name.toLowerCase().localeCompare(b.name.toLowerCase())
    ) || [];

  const { queryItems, updateQueryItems } = useCheckboxListStateManagement(
    sortedAvailableQueries,
    automatedQueryIds || []
  );

  const onSubmit = (evt: React.MouseEvent<HTMLFormElement> | KeyboardEvent) => {
    evt.preventDefault();

    const newQueryIds: number[] = [];
    queryItems?.forEach((p) => p.isChecked && newQueryIds.push(p.id));

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
                      <QueryFrequencyIndicator
                        frequency={interval}
                        checked={isChecked}
                      />
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
            <LogDestinationIndicator logDestination={logDestination} />
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
          <p>Automations currently run on macOS, Windows, and Linux hosts.</p>
          <p>
            Interested in query automations for your Chromebooks? &nbsp;
            <CustomLink
              url="https://fleetdm.com/contact"
              text="Let us know"
              newTab
            />
          </p>
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
