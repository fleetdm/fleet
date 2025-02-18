import React, { useState, useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { filter, includes } from "lodash";
import { InjectedRouter } from "react-router";

import { TAGGED_TEMPLATES } from "utilities/helpers";

import PATHS from "router/paths";

import permissions from "utilities/permissions";

import { AppContext } from "context/app";
import { QueryContext } from "context/query";

import queryAPI from "services/entities/queries";

// @ts-ignore
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import DataError from "components/DataError";

import {
  IListQueriesResponse,
  IQueryKeyQueriesLoadAll,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import { API_ALL_TEAMS_ID } from "interfaces/team";
import { DEFAULT_TARGETS_BY_TYPE } from "interfaces/target";

export interface ISelectQueryModalProps {
  onCancel: () => void;
  isOnlyObserver?: boolean;
  hostId: number;
  hostTeamId: number | null;
  router: InjectedRouter; // v3
  currentTeamId: number | undefined;
}

const baseClass = "select-query-modal";

const SelectQueryModal = ({
  onCancel,
  isOnlyObserver,
  hostId,
  hostTeamId,
  router,
  currentTeamId,
}: ISelectQueryModalProps): JSX.Element => {
  const { setSelectedQueryTargetsByType } = useContext(QueryContext);

  const { data: queries, error: queriesErr } = useQuery<
    IListQueriesResponse,
    Error,
    ISchedulableQuery[],
    IQueryKeyQueriesLoadAll[]
  >(
    [
      {
        scope: "queries",
        teamId: hostTeamId || API_ALL_TEAMS_ID,
        mergeInherited: hostTeamId !== API_ALL_TEAMS_ID,
      },
    ],
    ({ queryKey }) => queryAPI.loadAll(queryKey[0]),
    {
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      retry: false,
      select: (data: IListQueriesResponse) => data.queries,
    }
  );

  const onQueryHostCustom = () => {
    setSelectedQueryTargetsByType(DEFAULT_TARGETS_BY_TYPE);
    router.push(
      PATHS.NEW_QUERY() +
        TAGGED_TEMPLATES.queryByHostRoute(hostId, currentTeamId)
    );
  };

  const onQueryHostSaved = (selectedQuery: ISchedulableQuery) => {
    setSelectedQueryTargetsByType(DEFAULT_TARGETS_BY_TYPE);
    router.push(
      PATHS.EDIT_QUERY(selectedQuery.id) +
        TAGGED_TEMPLATES.queryByHostRoute(hostId, currentTeamId)
    );
  };

  let queriesAvailableToRun = queries;

  const { currentUser, isObserverPlus } = useContext(AppContext);

  /*  Context team id might be different that host's team id
  Observer plus must be checked against host's team id  */
  const isHostsTeamObserverPlus = currentUser
    ? permissions.isObserverPlus(currentUser, hostTeamId)
    : false;

  const [queriesFilter, setQueriesFilter] = useState("");

  if (isOnlyObserver && !isObserverPlus && !isHostsTeamObserverPlus) {
    queriesAvailableToRun =
      queries?.filter((query) => query.observer_can_run === true) || [];
  }

  const getQueries = () => {
    if (!queriesFilter) {
      return queriesAvailableToRun;
    }

    const lowerQueryFilter = queriesFilter.toLowerCase();

    return filter(queriesAvailableToRun, (query) => {
      if (!query.name) {
        return false;
      }

      const lowerQueryName = query.name.toLowerCase();

      return includes(lowerQueryName, lowerQueryFilter);
    });
  };

  const onFilterQueries = useCallback(
    (filterString: string): void => {
      setQueriesFilter(filterString);
    },
    [setQueriesFilter]
  );

  const queriesFiltered = getQueries();

  const queriesCount = queriesFiltered?.length || 0;

  const renderDescription = (): JSX.Element => {
    return (
      <div className={`${baseClass}__description`}>
        Choose a query to run on this host
        {(!isOnlyObserver || isObserverPlus || isHostsTeamObserverPlus) && (
          <>
            {" "}
            or{" "}
            <Button variant="text-link" onClick={onQueryHostCustom}>
              create your own query
            </Button>
          </>
        )}
        .
      </div>
    );
  };

  const renderQueries = (): JSX.Element => {
    if (queriesErr) {
      return <DataError />;
    }

    if (!queriesFilter && queriesCount === 0) {
      return (
        <div className={`${baseClass}__no-queries`}>
          <span className="info__header">You have no saved queries.</span>
          <span className="info__data">
            Expecting to see queries? Try again in a few seconds as the system
            catches up.
          </span>
        </div>
      );
    }

    if (queriesCount > 0) {
      const queryList =
        queriesFiltered?.map((query) => {
          return (
            <Button
              key={query.id}
              variant="unstyled-modal-query"
              className={`${baseClass}__modal-query-button`}
              onClick={() => onQueryHostSaved(query)}
            >
              <>
                <span className="info__header">{query.name}</span>
                {query.description && (
                  <span className="info__data">{query.description}</span>
                )}
              </>
            </Button>
          );
        }) || [];

      return (
        <>
          <InputFieldWithIcon
            name="query-filter"
            onChange={onFilterQueries}
            placeholder="Filter queries"
            value={queriesFilter}
            autofocus
            iconSvg="search"
            iconPosition="start"
          />
          <div className={`${baseClass}__query-selection`}>{queryList}</div>
        </>
      );
    }

    if (queriesFilter && queriesCount === 0) {
      return (
        <>
          <div className={`${baseClass}__filter-queries`}>
            <InputFieldWithIcon
              name="query-filter"
              onChange={onFilterQueries}
              placeholder="Filter queries"
              value={queriesFilter}
              autofocus
              iconSvg="search"
              iconPosition="start"
            />
          </div>
          <div className={`${baseClass}__no-queries`}>
            <span className="info__header">
              No queries match the current search criteria.
            </span>
            <span className="info__data">
              Expecting to see queries? Try again in a few seconds as the system
              catches up.
            </span>
          </div>
        </>
      );
    }
    return <></>;
  };

  return (
    <Modal
      title="Select a query"
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
      width="large"
    >
      <>
        {renderDescription()}
        {renderQueries()}
      </>
    </Modal>
  );
};

export default SelectQueryModal;
