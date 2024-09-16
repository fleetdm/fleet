import React, { useState, useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner/InfoBanner";
import CustomLink from "components/CustomLink/CustomLink";
import Checkbox from "components/forms/fields/Checkbox/Checkbox";
import QueryFrequencyIndicator from "components/QueryFrequencyIndicator/QueryFrequencyIndicator";
import LogDestinationIndicator from "components/LogDestinationIndicator/LogDestinationIndicator";

import { ISchedulableQuery } from "interfaces/schedulable_query";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import { CONTACT_FLEET_LINK } from "utilities/constants";

interface IManageQueryAutomationsModalProps {
  isUpdatingAutomations: boolean;
  onSubmit: (formData: any) => void; // TODO
  onCancel: () => void;
  isShowingPreviewDataModal: boolean;
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

const baseClass = "manage-query-automations-modal";

const ManageQueryAutomationsModal = ({
  isUpdatingAutomations,
  automatedQueryIds,
  onSubmit,
  onCancel,
  isShowingPreviewDataModal,
  togglePreviewDataModal,
  availableQueries,
  logDestination,
}: IManageQueryAutomationsModalProps): JSX.Element => {
  // TODO: Error handling, if any
  // const [errors, setErrors] = useState<{ [key: string]: string }>({});

  // Client side sort queries alphabetically
  const sortedAvailableQueries =
    availableQueries?.sort((a, b) =>
      a.name.toLowerCase().localeCompare(b.name.toLowerCase())
    ) || [];

  const { queryItems, updateQueryItems } = useCheckboxListStateManagement(
    sortedAvailableQueries,
    automatedQueryIds || []
  );

  const onSubmitQueryAutomations = (
    evt: React.MouseEvent<HTMLFormElement> | KeyboardEvent
  ) => {
    evt.preventDefault();

    const newQueryIds: number[] = [];
    queryItems?.forEach((p) => p.isChecked && newQueryIds.push(p.id));

    onSubmit(newQueryIds);
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
      title="Manage automations"
      onExit={onCancel}
      className={baseClass}
      width="large"
      isHidden={isShowingPreviewDataModal}
    >
      <div className={`${baseClass} form`}>
        <div className={`${baseClass}__heading`}>
          Query automations let you send data to your log destination on a
          schedule. Data is sent according to a query’s frequency.
        </div>
        {availableQueries?.length ? (
          <div className={`${baseClass}__select form-field`}>
            <div className="form-field__label">
              Choose which queries will send data:
            </div>
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
                          // !isChecked &&
                          //   setErrors((errs) => omit(errs, "queryItems"));
                        }}
                      >
                        <TooltipTruncatedText value={name} />
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
        <div className={`${baseClass}__log-destination form-field`}>
          <div className="form-field__label">Log destination:</div>
          <div className={`${baseClass}__selection`}>
            <LogDestinationIndicator logDestination={logDestination} />
          </div>
          <div className={`${baseClass}__configure form-field__help-text`}>
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
            <CustomLink url={CONTACT_FLEET_LINK} text="Let us know" newTab />
          </p>
        </InfoBanner>
        <Button
          type="button"
          variant="text-link"
          onClick={togglePreviewDataModal}
          className={`${baseClass}__preview-data`}
        >
          Preview data
        </Button>
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            variant="brand"
            onClick={onSubmitQueryAutomations}
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
    </Modal>
  );
};

export default ManageQueryAutomationsModal;
