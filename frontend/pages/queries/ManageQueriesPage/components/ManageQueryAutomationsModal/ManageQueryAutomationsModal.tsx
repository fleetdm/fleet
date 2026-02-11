import React, { useState, useEffect, useContext, useMemo } from "react";
import { useQuery } from "react-query";

import { AppContext } from "context/app";

import { IQueryKeyQueriesLoadAll } from "interfaces/schedulable_query";
import { LogDestination } from "interfaces/config";
import queriesAPI, { IQueriesResponse } from "services/entities/queries";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink/CustomLink";
import Checkbox from "components/forms/fields/Checkbox/Checkbox";
import QueryFrequencyIndicator from "components/QueryFrequencyIndicator/QueryFrequencyIndicator";
import LogDestinationIndicator from "components/LogDestinationIndicator/LogDestinationIndicator";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

export interface IQueryAutomationsSubmitData {
  newAutomatedQueryIds: number[];
  previousAutomatedQueryIds: number[];
}

interface IManageQueryAutomationsModalProps {
  isUpdatingAutomations: boolean;
  onSubmit: (formData: IQueryAutomationsSubmitData) => void;
  onCancel: () => void;
  isShowingPreviewDataModal: boolean;
  togglePreviewDataModal: () => void;
  teamId?: number;
  logDestination: LogDestination;
  webhookDestination?: string;
  filesystemDestination?: string;
}

interface ICheckedQuery {
  name?: string;
  id: number;
  isChecked: boolean;
  interval: number;
}

const baseClass = "manage-query-automations-modal";

const ManageQueryAutomationsModal = ({
  isUpdatingAutomations,
  onSubmit,
  onCancel,
  isShowingPreviewDataModal,
  togglePreviewDataModal,
  teamId,
  logDestination,
  webhookDestination,
  filesystemDestination,
}: IManageQueryAutomationsModalProps): JSX.Element => {
  const gitOpsModeEnabled = useContext(AppContext).config?.gitops
    .gitops_mode_enabled;

  // Fetch team-scoped queries (mergeInherited: false) so we only show
  // queries that belong to this team, not inherited global queries.
  const { data: queriesResponse } = useQuery<
    IQueriesResponse,
    Error,
    IQueriesResponse,
    IQueryKeyQueriesLoadAll[]
  >(
    [
      {
        scope: "queries",
        teamId,
        orderKey: "name",
        orderDirection: "asc" as const,
        mergeInherited: false,
      },
    ],
    ({ queryKey }) => queriesAPI.loadAll(queryKey[0]),
    {
      refetchOnWindowFocus: false,
    }
  );

  const availableQueries = useMemo(() => queriesResponse?.queries ?? [], [
    queriesResponse,
  ]);

  const automatedQueryIds = useMemo(
    () =>
      availableQueries
        .filter((query) => query.automations_enabled)
        .map((query) => query.id),
    [availableQueries]
  );

  // Client side sort queries alphabetically
  const sortedAvailableQueries = useMemo(
    () =>
      [...availableQueries].sort((a, b) =>
        a.name.toLowerCase().localeCompare(b.name.toLowerCase())
      ),
    [availableQueries]
  );

  const [queryItems, setQueryItems] = useState<ICheckedQuery[]>([]);

  // Sync queryItems when the async fetch completes.
  useEffect(() => {
    if (sortedAvailableQueries.length > 0) {
      setQueryItems(
        sortedAvailableQueries.map(({ name, id, interval }) => ({
          name,
          id,
          isChecked: !!automatedQueryIds?.includes(id),
          interval,
        }))
      );
    }
  }, [sortedAvailableQueries, automatedQueryIds]);

  const updateQueryItems = (queryId: number) => {
    setQueryItems((prevItems) =>
      prevItems.map((query) =>
        query.id !== queryId ? query : { ...query, isChecked: !query.isChecked }
      )
    );
  };

  const onSubmitQueryAutomations = (
    evt: React.MouseEvent<HTMLFormElement> | KeyboardEvent
  ) => {
    evt.preventDefault();

    const newQueryIds: number[] = [];
    queryItems?.forEach((p) => p.isChecked && newQueryIds.push(p.id));

    onSubmit({
      newAutomatedQueryIds: newQueryIds,
      previousAutomatedQueryIds: automatedQueryIds,
    });
  };

  useEffect(() => {
    const listener = (event: KeyboardEvent) => {
      if (event.code === "Enter" || event.code === "NumpadEnter") {
        event.preventDefault();
        onSubmit({
          newAutomatedQueryIds: queryItems
            .filter((p) => p.isChecked)
            .map((p) => p.id),
          previousAutomatedQueryIds: automatedQueryIds,
        });
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
          Report automations let you send data gathered from macOS, Windows, and
          Linux hosts to a log destination. Data is sent according to a
          report&apos;s interval.
        </div>
        {availableQueries.length ? (
          <div className={`${baseClass}__select form-field`}>
            <div className="form-field__label">
              Choose which reports will send data:
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
                        }}
                        disabled={gitOpsModeEnabled}
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
            <b>You have no reports.</b>
            <p>Add a report to turn on automations.</p>
          </div>
        )}
        <div className={`${baseClass}__log-destination form-field`}>
          <div className="form-field__label">Log destination:</div>
          <div className={`${baseClass}__selection`}>
            <LogDestinationIndicator
              logDestination={logDestination}
              webhookDestination={webhookDestination}
              filesystemDestination={filesystemDestination}
            />
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
        <Button
          type="button"
          variant="inverse"
          onClick={togglePreviewDataModal}
          className={`${baseClass}__preview-data`}
        >
          Preview data
        </Button>
        <div className="modal-cta-wrap">
          <GitOpsModeTooltipWrapper
            tipOffset={6}
            renderChildren={(disableChildren) => (
              <Button
                type="submit"
                onClick={onSubmitQueryAutomations}
                className="save-loading"
                isLoading={isUpdatingAutomations}
                disabled={disableChildren}
              >
                Save
              </Button>
            )}
          />
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ManageQueryAutomationsModal;
