import React, { useState, useCallback } from "react"; //, { useEffect }
import { useDispatch, useSelector } from "react-redux";
// @ts-ignore
import memoize from "memoize-one";

import { push } from "react-router-redux";
import { IQuery } from "interfaces/query";
import { IGlobalScheduledQuery } from "interfaces/global_scheduled_query";
// @ts-ignore
import globalScheduledQueryActions from "redux/nodes/entities/global_scheduled_queries/actions";

import paths from "router/paths";
import Button from "components/buttons/Button";
import ScheduleError from "./components/ScheduleError";
import ScheduleListWrapper from "./components/ScheduleListWrapper";
import ScheduleEditorModal from "./components/ScheduleEditorModal";
import RemoveScheduledQueryModal from "./components/RemoveScheduledQueryModal";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

const baseClass = "manage-schedule-page";

// FAKE DATA ALERT

const fakeData = {
  scheduled: [
    {
      id: 1,
      query_id: 4,
      query_name: "Get crashes",
      interval: 172800,
      last_executed: "2021-06-23T20:26:51Z",
    },
    {
      id: 2,
      query_id: 7,
      query_name: "Detect machines with Gatekeeper disabled",
      interval: 14400,
      last_executed: "2021-06-24T20:26:51Z",
    },
    {
      id: 3,
      query_id: 8,
      query_name: "Detect fake data",
      interval: 86400,
      last_executed: "2021-06-23T20:26:51Z",
    },
    {
      id: 4,
      query_id: 20,
      query_name: "Detect a shit ton of work",
      interval: 604800,
      last_executed: "2021-06-21T20:26:51Z",
    },
  ],
};

// 1. SET TO TRUE IF YOU WANT TO SEE THE ERROR STATE RENDER;
const fakeDataError = false;

// END FAKE DATA ALERT

const renderTable = (toggleRemoveScheduledQueryModal: any): JSX.Element => {
  // Schedule has an error retrieving data.
  if (fakeDataError) {
    return <ScheduleError />;
  }

  // Schedule table
  return (
    <ScheduleListWrapper
      fakeData={fakeData}
      toggleRemoveScheduledQueryModal={toggleRemoveScheduledQueryModal}
    />
  );
};
interface IFetchParams {
  pageIndex?: number;
  pageSize?: number;
  searchQuery?: string;
}
interface IRootState {
  entities: {
    global_scheduled_queries: {
      isLoading: boolean;
      data: IGlobalScheduledQuery[];
    };
    queries: {
      isLoading: boolean;
      data: IQuery[];
    };
  };
}

// const getQueries = (data: { [id: string]: IGlobalScheduledQuery }) => {
//   return Object.keys(data).map((queryId) => {
//     return data[queryId];
//   });
// };
// const memoizedGetQueries = memoize(getQueries);
// console.log("memoizedGetQueries", memoizedGetQueries);

const ManageSchedulePage = (): JSX.Element => {
  const dispatch = useDispatch();
  const { MANAGE_PACKS } = paths;
  const handleAdvanced = (event: any) => dispatch(push(MANAGE_PACKS));

  dispatch(globalScheduledQueryActions.loadAll());

  const [showScheduleEditorModal, setShowScheduleEditorModal] = useState(false);
  const [
    showRemoveScheduledQueryModal,
    setShowRemoveScheduledQueryModal,
  ] = useState(false);
  const [selectedQueryIds, setSelectedQueryIds] = useState([]);

  const toggleScheduleEditorModal = useCallback(() => {
    setShowScheduleEditorModal(!showScheduleEditorModal);
  }, [showScheduleEditorModal, setShowScheduleEditorModal]);

  const toggleRemoveScheduledQueryModal = useCallback(
    (queryIds?) => {
      setShowRemoveScheduledQueryModal(!showRemoveScheduledQueryModal);
      setSelectedQueryIds(queryIds); // haven't tested this
    },
    [
      showRemoveScheduledQueryModal,
      setShowRemoveScheduledQueryModal,
      setSelectedQueryIds,
    ]
  );

  // TODO: Figure out correct removal once queries are populated in
  const onRemoveScheduledQuerySubmit = useCallback(() => {
    console.log("onRemoveScheduleQuerySubmit fires after click!");
    // Get selectedQueryIds from state
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

  // TODO: Fix all queries to load always
  // This is all queries for the ScheduleEditorModal
  const loadingAllQueriesData = useSelector(
    (state: IRootState) => state.entities.queries.isLoading
  );
  const allQueries = useSelector(
    (state: IRootState) => state.entities.queries.data
  );
  // Turn object of objects into array of objects
  const allQueriesList = Object.values(allQueries);
  console.log("allQueriesList", allQueriesList);

  // TODO: Fix formData remove not passing correctly
  const onAddScheduledQuerySubmit = useCallback(
    (formData: any) => {
      console.log(
        "onAddScheduledQuerySubmit globalScheduledQueryActions.create(formData....... is...",
        formData
      );
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
            renderFlash("error", "Could not schedule query. Please try again.")
          );
        });
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
          {fakeData.scheduled.length !== 0 && !fakeDataError && (
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
        <div>{renderTable(toggleRemoveScheduledQueryModal)}</div>
        {showScheduleEditorModal && (
          <ScheduleEditorModal
            onCancel={toggleScheduleEditorModal}
            onScheduleSubmit={onAddScheduledQuerySubmit}
            allQueries={allQueriesList}
          />
        )}
        {showRemoveScheduledQueryModal && (
          <RemoveScheduledQueryModal
            onCancel={toggleRemoveScheduledQueryModal}
            onSubmit={onRemoveScheduledQuerySubmit}
            selectedQueryIds={selectedQueryIds}
          />
        )}
      </div>
    </div>
  );
};

export default ManageSchedulePage;
