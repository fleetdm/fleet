import React, { useState, useCallback, useContext } from "react";

import { filter, includes } from "lodash";
import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
// @ts-ignore
import InputFieldWithIcon from "components/forms/fields/InputFieldWithIcon";

import DataError from "components/DataError";
import permissions from "utilities/permissions";
import { ISchedulableQuery } from "interfaces/schedulable_query";

export interface ISelectQueryModalProps {
  onCancel: () => void;
  onQueryHostCustom: () => void;
  onQueryHostSaved: (selectedQuery: ISchedulableQuery) => void;
  queries: ISchedulableQuery[] | [];
  queryErrors: Error | null;
  isOnlyObserver?: boolean;
  hostsTeamId: number | null;
}

const baseClass = "select-query-modal";

const SelectQueryModal = ({
  onCancel,
  onQueryHostCustom,
  onQueryHostSaved,
  queries,
  queryErrors,
  isOnlyObserver,
  hostsTeamId,
}: ISelectQueryModalProps): JSX.Element => {
  let queriesAvailableToRun = queries;

  const { currentUser, isObserverPlus } = useContext(AppContext);

  /*  Context team id might be different that host's team id
  Observer plus must be checked against host's team id  */
  const isHostsTeamObserverPlus = currentUser
    ? permissions.isObserverPlus(currentUser, hostsTeamId)
    : false;

  const [queriesFilter, setQueriesFilter] = useState("");

  if (isOnlyObserver && !isObserverPlus && !isHostsTeamObserverPlus) {
    queriesAvailableToRun = queries.filter(
      (query) => query.observer_can_run === true
    );
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

  const queriesCount = queriesFiltered.length;

  const customQueryButton = () => {
    return (
      <Button
        onClick={() => onQueryHostCustom()}
        variant="brand"
        className={`${baseClass}__custom-query-button`}
      >
        Create custom query
      </Button>
    );
  };

  const results = (): JSX.Element => {
    if (queryErrors) {
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
          <div className="modal-cta-wrap">
            {(!isOnlyObserver || isObserverPlus || isHostsTeamObserverPlus) &&
              customQueryButton()}
          </div>
        </div>
      );
    }

    if (queriesCount > 0) {
      const queryList = queriesFiltered.map((query) => {
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
      });
      return (
        <div>
          <div className={`${baseClass}__filter-create-wrapper`}>
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
            {(!isOnlyObserver || isObserverPlus || isHostsTeamObserverPlus) && (
              <div className={`${baseClass}__create-query`}>
                <span>OR</span>
                {customQueryButton()}
              </div>
            )}
          </div>
          <div>{queryList}</div>
        </div>
      );
    }

    if (queriesFilter && queriesCount === 0) {
      return (
        <div>
          <div className={`${baseClass}__filter-create-wrapper`}>
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
            {(!isOnlyObserver || isObserverPlus || isHostsTeamObserverPlus) && (
              <div className={`${baseClass}__create-query`}>
                <span>OR</span>
                {customQueryButton()}
              </div>
            )}
          </div>
          <div className={`${baseClass}__no-query-results`}>
            <span className="info__header">
              No queries match the current search criteria.
            </span>
            <span className="info__data">
              Expecting to see queries? Try again in a few seconds as the system
              catches up.
            </span>
          </div>
        </div>
      );
    }
    return <></>;
  };

  return (
    <Modal
      title="Select a query"
      onExit={onCancel}
      className={`${baseClass}__modal`}
      width="xlarge"
    >
      {results()}
    </Modal>
  );
};

export default SelectQueryModal;
