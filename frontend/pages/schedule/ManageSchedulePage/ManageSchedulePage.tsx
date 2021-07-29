import React, { useState, useCallback, useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";

import { push } from "react-router-redux";
import { IQuery } from "interfaces/query";
import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";
// @ts-ignore
import globalScheduledQueryActions from "redux/nodes/entities/global_scheduled_queries/actions";
// @ts-ignore
import queryActions from "redux/nodes/entities/queries/actions";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import paths from "router/paths";
import Button from "components/buttons/Button";
import ScheduleError from "./components/ScheduleError";
import ScheduleListWrapper from "./components/ScheduleListWrapper";
import ScheduleEditorModal from "./components/ScheduleEditorModal";
import RemoveScheduledQueryModal from "./components/RemoveScheduledQueryModal";

const baseClass = "manage-schedule-page";

const renderTable = (
  onRemoveScheduledQueryClick: React.MouseEventHandler<HTMLButtonElement>,
  onEditScheduledQueryClick: React.MouseEventHandler<HTMLButtonElement>,
  allGlobalScheduledQueriesList: IGlobalScheduledQuery[],
  allGlobalScheduledQueriesError: any,
  toggleScheduleEditorModal: () => void
): JSX.Element => {
  if (Object.keys(allGlobalScheduledQueriesError).length !== 0) {
    return <ScheduleError />;
  }

  return (
    <ScheduleListWrapper
      onRemoveScheduledQueryClick={onRemoveScheduledQueryClick}
      onEditScheduledQueryClick={onEditScheduledQueryClick}
      allGlobalScheduledQueriesList={allGlobalScheduledQueriesList}
      toggleScheduleEditorModal={toggleScheduleEditorModal}
    />
  );
};
interface IRootState {
  entities: {
    global_scheduled_queries: {
      isLoading: boolean;
      data: IGlobalScheduledQuery[];
      errors: any;
    };
    queries: {
      isLoading: boolean;
      data: IQuery[];
    };
  };
}
interface IFormData {
  interval: number;
  name?: string;
  shard: number;
  query?: string;
  query_id?: number;
  logging_type: string;
  platform: string;
  version: string;
}

const ManageSchedulePage = (): JSX.Element => {
  const dispatch = useDispatch();
  const { MANAGE_PACKS } = paths;
  const handleAdvanced = () => dispatch(push(MANAGE_PACKS));

  useEffect(() => {
    dispatch(globalScheduledQueryActions.loadAll());
    dispatch(queryActions.loadAll());
  }, [dispatch]);

  const allQueries = useSelector((state: IRootState) => state.entities.queries);
  const allQueriesList = Object.values(allQueries.data);

  const allGlobalScheduledQueries = useSelector(
    (state: IRootState) => state.entities.global_scheduled_queries
  );
  const allGlobalScheduledQueriesList = Object.values(
    allGlobalScheduledQueries.data
  );
  const allGlobalScheduledQueriesError = allGlobalScheduledQueries.errors;

  const [showScheduleEditorModal, setShowScheduleEditorModal] = useState(false);
  const [
    showRemoveScheduledQueryModal,
    setShowRemoveScheduledQueryModal,
  ] = useState(false);
  const [selectedQueryIds, setSelectedQueryIds] = useState([]);
  const [
    selectedScheduledQuery,
    setSelectedScheduledQuery,
  ] = useState<IGlobalScheduledQuery>();

  const toggleScheduleEditorModal = useCallback(() => {
    setSelectedScheduledQuery(undefined); // create modal renders
    setShowScheduleEditorModal(!showScheduleEditorModal);
  }, [showScheduleEditorModal, setShowScheduleEditorModal]);

  const toggleRemoveScheduledQueryModal = useCallback(() => {
    setShowRemoveScheduledQueryModal(!showRemoveScheduledQueryModal);
  }, [showRemoveScheduledQueryModal, setShowRemoveScheduledQueryModal]);

  const onRemoveScheduledQueryClick = (selectedTableQueryIds: any): any => {
    toggleRemoveScheduledQueryModal();
    setSelectedQueryIds(selectedTableQueryIds);
  };

  const onEditScheduledQueryClick = (selectedQuery: any): any => {
    toggleScheduleEditorModal();
    setSelectedScheduledQuery(selectedQuery); // edit modal renders
  };

  const onRemoveScheduledQuerySubmit = useCallback(() => {
    const promises = selectedQueryIds.map((id: number) => {
      return dispatch(globalScheduledQueryActions.destroy({ id }));
    });
    const queryOrQueries = selectedQueryIds.length === 1 ? "query" : "queries";
    return Promise.all(promises)
      .then(() => {
        dispatch(
          renderFlash(
            "success",
            `Successfully removed scheduled ${queryOrQueries}.`
          )
        );
        toggleRemoveScheduledQueryModal();
        dispatch(globalScheduledQueryActions.loadAll());
      })
      .catch(() => {
        dispatch(
          renderFlash(
            "error",
            `Unable to remove scheduled ${queryOrQueries}. Please try again.`
          )
        );
        toggleRemoveScheduledQueryModal();
      });
  }, [dispatch, selectedQueryIds, toggleRemoveScheduledQueryModal]);

  const onAddScheduledQuerySubmit = useCallback(
    (formData: IFormData, update: boolean) => {
      if (update) {
        console.log("Sweet you made it to update!");
        // dispatch(globalScheduledQueryActions.update({ ...formData }))
        // .then(() => {
        //   dispatch(
        //     renderFlash(
        //       "success",
        //       `Successfully added ${formData.name} to the schedule.`
        //     )
        //   );
        //   dispatch(globalScheduledQueryActions.loadAll());
        // })
        // .catch(() => {
        //   dispatch(
        //     renderFlash("error", "Could not schedule query. Please try again.")
        //   );
        // });
      } else {
        dispatch(globalScheduledQueryActions.create({ ...formData }))
          .then(() => {
            dispatch(
              renderFlash(
                "success",
                `Successfully added ${formData.name} to the schedule.`
              )
            );
            dispatch(globalScheduledQueryActions.loadAll());
          })
          .catch(() => {
            dispatch(
              renderFlash(
                "error",
                "Could not schedule query. Please try again."
              )
            );
          });
      }
      toggleScheduleEditorModal();
    },
    [dispatch, toggleScheduleEditorModal]
  );

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <h1 className={`${baseClass}__title`}>
                <span>Schedule</span>
              </h1>
              <div className={`${baseClass}__description`}>
                <p>
                  Schedule recurring queries for your hosts. Fleet’s query
                  schedule lets you add queries which are executed at regular
                  intervals.
                </p>
              </div>
            </div>
          </div>
          {/* Hide CTA Buttons if no schedule or schedule error */}
          {allGlobalScheduledQueriesList.length !== 0 &&
            allGlobalScheduledQueriesError.length !== 0 && (
              <div className={`${baseClass}__action-button-container`}>
                <Button
                  variant="inverse"
                  onClick={handleAdvanced}
                  className={`${baseClass}__advanced-button`}
                >
                  Advanced
                </Button>
                <Button
                  variant="brand"
                  className={`${baseClass}__schedule-button`}
                  onClick={toggleScheduleEditorModal}
                >
                  Schedule a query
                </Button>
              </div>
            )}
        </div>
        <div>
          {renderTable(
            onRemoveScheduledQueryClick,
            onEditScheduledQueryClick,
            allGlobalScheduledQueriesList,
            allGlobalScheduledQueriesError,
            toggleScheduleEditorModal
          )}
        </div>
        {showScheduleEditorModal && (
          <ScheduleEditorModal
            onCancel={toggleScheduleEditorModal}
            onScheduleSubmit={onAddScheduledQuerySubmit}
            allQueries={allQueriesList}
            selectedScheduledQuery={selectedScheduledQuery}
          />
        )}
        {showRemoveScheduledQueryModal && (
          <RemoveScheduledQueryModal
            onCancel={toggleRemoveScheduledQueryModal}
            onSubmit={onRemoveScheduledQuerySubmit}
          />
        )}
      </div>
    </div>
  );
};

export default ManageSchedulePage;
