import React, { useState, useCallback } from "react";

import { filter, includes } from "lodash";

import { IHost } from "interfaces/host";
import { IQuery } from "interfaces/query";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

import OpenNewTabIcon from "../../../../../assets/images/open-new-tab-12x12@2x.png";
import ErrorIcon from "../../../../../assets/images/icon-error-16x16@2x.png";

export interface ISelectQueryModalProps {
  host: IHost;
  onCancel: () => void;
  onQueryHostCustom: () => void;
  onQueryHostSaved: (selectedQuery: IQuery) => void;
  queries: IQuery[] | [];
  queryErrors: any | null;
  isOnlyObserver: boolean | undefined;
}

const baseClass = "select-query-modal";

const SelectQueryModal = ({
  onCancel,
  onQueryHostCustom,
  onQueryHostSaved,
  queries,
  queryErrors,
  isOnlyObserver,
}: ISelectQueryModalProps) => {
  let queriesAvailableToRun = queries;

  const [queriesFilter, setQueriesFilter] = useState("");

  if (isOnlyObserver) {
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
      return (
        <div className={`${baseClass}__no-queries`}>
          <span className="info__header">
            <img src={ErrorIcon} alt="error icon" id="error-icon" />
            Something&apos;s gone wrong.
          </span>
          <span className="info__data">Refresh the page or log in again.</span>
          <span className="info__data">
            If this keeps happening, please&nbsp;
            <a
              href="https://github.com/fleetdm/fleet/issues"
              target="_blank"
              rel="noopener noreferrer"
            >
              file an issue
              <img src={OpenNewTabIcon} alt="open new tab" id="new-tab-icon" />
            </a>
          </span>
          {!isOnlyObserver && customQueryButton()}
        </div>
      );
    }

    if (!queriesFilter && queriesCount === 0) {
      return (
        <div className={`${baseClass}__no-queries`}>
          <span className="info__header">You have no saved queries.</span>
          <span className="info__data">
            Expecting to see queries? Try again in a few seconds as the system
            catches up.
          </span>
          {!isOnlyObserver && customQueryButton()}
        </div>
      );
    }

    if (queriesCount > 0) {
      const queryList = queriesFiltered.map((query) => {
        return (
          <Button
            key={query.id}
            variant="unstyled-modal-query"
            className="modal-query-button"
            onClick={() => onQueryHostSaved(query)}
          >
            <>
              <span className="info__header">{query.name}</span>
              <span className="info__data">{query.description}</span>
            </>
          </Button>
        );
      });
      return (
        <div>
          <div className={`${baseClass}__query-modal`}>
            <div className={`${baseClass}__filter-queries`}>
              <InputField
                name="query-filter"
                onChange={onFilterQueries}
                placeholder="Filter queries"
                value={queriesFilter}
              />
            </div>
            {!isOnlyObserver && (
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
          <div className={`${baseClass}__query-modal`}>
            <div className={`${baseClass}__filter-queries`}>
              <InputField
                name="query-filter"
                onChange={onFilterQueries}
                placeholder="Filter queries"
                value={queriesFilter}
              />
            </div>
            {!isOnlyObserver && (
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
    >
      {results()}
    </Modal>
  );
};

export default SelectQueryModal;
