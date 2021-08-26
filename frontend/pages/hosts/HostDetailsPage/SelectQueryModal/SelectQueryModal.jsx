import React, { useState } from "react";
import PropTypes from "prop-types";

import { filter, includes } from "lodash";

import Button from "components/buttons/Button";
import Modal from "components/modals/Modal";
import InputField from "components/forms/fields/InputField";

import queryInterface from "interfaces/query";
import hostInterface from "interfaces/host";

import helpers from "../helpers";

import OpenNewTabIcon from "../../../../../assets/images/open-new-tab-12x12@2x.png";
import ErrorIcon from "../../../../../assets/images/icon-error-16x16@2x.png";

const baseClass = "select-query-modal";

const onQueryHostCustom = (host, dispatch) => {
  const { queryHostCustom } = helpers;

  queryHostCustom(dispatch, host);

  return false;
};

const onQueryHostSaved = (host, selectedQuery, dispatch) => {
  const { queryHostSaved } = helpers;

  queryHostSaved(dispatch, host, selectedQuery);

  return false;
};

const SelectQueryModal = (props) => {
  const { host, onCancel, dispatch, queries, queryErrors } = props;

  const [queriesFilter, setQueriesFilter] = useState("");

  const getQueries = () => {
    if (!queriesFilter) {
      return queries;
    }

    const lowerQueryFilter = queriesFilter.toLowerCase();

    return filter(queries, (query) => {
      if (!query.name) {
        return false;
      }

      const lowerQueryName = query.name.toLowerCase();

      return includes(lowerQueryName, lowerQueryFilter);
    });
  };

  const customQueryButton = () => {
    return (
      <Button
        onClick={() => onQueryHostCustom(host, dispatch)}
        variant="brand"
        className={`${baseClass}__custom-query-button`}
      >
        Create custom query
      </Button>
    );
  };

  const onFilterQueries = (event) => {
    setQueriesFilter(event);
    return false;
  };

  const queriesFiltered = getQueries();

  const queriesCount = queriesFiltered.length;

  const results = () => {
    if (queryErrors) {
      return (
        <div className={`${baseClass}__no-queries`}>
          <span className="info__header">
            <img src={ErrorIcon} alt="error icon" id="error-icon" />
            Something&apos;s gone wrong.
          </span>
          <span className="info__data">Refresh the page or log in again.</span>
          <span className="info__data">
            If this keeps happening, please{" "}
            <a
              href="https://github.com/fleetdm/fleet/issues"
              target="_blank"
              rel="noopener noreferrer"
            >
              file an issue.
              <img src={OpenNewTabIcon} alt="open new tab" id="new-tab-icon" />
            </a>
          </span>
          {customQueryButton()}
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
          {customQueryButton()}
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
            onClick={() => onQueryHostSaved(host, query, dispatch)}
          >
            <span className="info__header">{query.name}</span>
            <span className="info__data">{query.description}</span>
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
            <div className={`${baseClass}__create-query`}>
              <span>OR</span>
              {customQueryButton()}
            </div>
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
            <div className={`${baseClass}__create-query`}>
              <span>OR</span>
              {customQueryButton()}
            </div>
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
    return null;
  };

  return (
    <Modal
      title="Select a query"
      onExit={onCancel(null)}
      className={`${baseClass}__modal`}
    >
      {results()}
    </Modal>
  );
};

SelectQueryModal.propTypes = {
  dispatch: PropTypes.func,
  host: hostInterface,
  queries: PropTypes.arrayOf(queryInterface),
  onCancel: PropTypes.func,
  queryErrors: PropTypes.object, // eslint-disable-line react/forbid-prop-types
};

export default SelectQueryModal;
