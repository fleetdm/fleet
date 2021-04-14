import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { filter, get, includes, pull } from "lodash";
import { push } from "react-router-redux";

import Button from "components/buttons/Button";
import entityGetter from "redux/utilities/entityGetter";
import InputField from "components/forms/fields/InputField";
import Modal from "components/modals/Modal";
import KolideIcon from "components/icons/KolideIcon";
import SecondarySidePanelContainer from "components/side_panels/SecondarySidePanelContainer";
import PATHS from "router/paths";
import QueryDetailsSidePanel from "components/side_panels/QueryDetailsSidePanel";
import QueriesList from "components/queries/QueriesList";
import queryActions from "redux/nodes/entities/queries/actions";
import queryInterface from "interfaces/query";
import { renderFlash } from "redux/nodes/notifications/actions";

const baseClass = "manage-queries-page";

export class ManageQueriesPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    loadingQueries: PropTypes.bool.isRequired,
    queries: PropTypes.arrayOf(queryInterface),
    selectedQuery: queryInterface,
  };

  static defaultProps = {
    dispatch: () => Promise.resolve(),
    loadingQueries: false,
  };

  constructor(props) {
    super(props);

    this.state = {
      allQueriesChecked: false,
      checkedQueryIDs: [],
      queriesFilter: "",
      showModal: false,
    };
  }

  componentWillMount() {
    const { dispatch } = this.props;

    dispatch(queryActions.loadAll()).catch(() => false);

    return false;
  }

  onDeleteQueries = (evt) => {
    evt.preventDefault();

    const { checkedQueryIDs } = this.state;
    const { dispatch } = this.props;
    const { destroy } = queryActions;

    const promises = checkedQueryIDs.map((queryID) => {
      return dispatch(destroy({ id: queryID }));
    });

    return Promise.all(promises)
      .then(() => {
        dispatch(renderFlash("success", "Queries successfully deleted."));

        this.setState({ checkedQueryIDs: [], showModal: false });

        return false;
      })
      .catch(() => {
        dispatch(renderFlash("error", "Something went wrong."));

        this.setState({ showModal: false });

        return false;
      });
  };

  onCheckAllQueries = (shouldCheck) => {
    if (shouldCheck) {
      const queries = this.getQueries();
      const checkedQueryIDs = queries.map((query) => query.id);

      this.setState({ allQueriesChecked: true, checkedQueryIDs });

      return false;
    }

    this.setState({ allQueriesChecked: false, checkedQueryIDs: [] });

    return false;
  };

  onCheckQuery = (checked, id) => {
    const { checkedQueryIDs } = this.state;
    const newCheckedQueryIDs = checked
      ? checkedQueryIDs.concat(id)
      : pull(checkedQueryIDs, id);

    this.setState({
      allQueriesChecked: false,
      checkedQueryIDs: newCheckedQueryIDs,
    });

    return false;
  };

  onFilterQueries = (queriesFilter) => {
    this.setState({ queriesFilter });

    return false;
  };

  onSelectQuery = (selectedQuery) => {
    const { dispatch } = this.props;
    const locationObject = {
      pathname: PATHS.MANAGE_QUERIES,
      query: { selectedQuery: selectedQuery.id },
    };

    dispatch(push(locationObject));

    return false;
  };

  onDblClickQuery = (selectedQuery) => {
    const { dispatch } = this.props;

    dispatch(push(PATHS.EDIT_QUERY(selectedQuery)));

    return false;
  };

  onToggleModal = () => {
    const { showModal } = this.state;

    this.setState({ showModal: !showModal });

    return false;
  };

  getQueries = () => {
    const { queriesFilter } = this.state;
    const { queries } = this.props;

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

  goToNewQueryPage = () => {
    const { dispatch } = this.props;
    const { NEW_QUERY } = PATHS;

    dispatch(push(NEW_QUERY));

    return false;
  };

  goToEditQueryPage = (query) => {
    const { dispatch } = this.props;
    const { EDIT_QUERY } = PATHS;

    dispatch(push(EDIT_QUERY(query)));

    return false;
  };

  renderCTAs = () => {
    const { goToNewQueryPage, onToggleModal } = this;
    const btnClass = `${baseClass}__delete-queries-btn`;
    const checkedQueryCount = this.state.checkedQueryIDs.length;

    if (checkedQueryCount) {
      const queryText = checkedQueryCount === 1 ? "Query" : "Queries";

      return (
        <div className={`${baseClass}__ctas`}>
          <p className={`${baseClass}__query-count`}>
            {checkedQueryCount} {queryText} selected
          </p>
          <Button className={btnClass} onClick={onToggleModal} variant="alert">
            Delete
          </Button>
        </div>
      );
    }

    return (
      <Button variant="brand" onClick={goToNewQueryPage}>
        Create new query
      </Button>
    );
  };

  renderModal = () => {
    const { onDeleteQueries, onToggleModal } = this;
    const { showModal } = this.state;

    if (!showModal) {
      return false;
    }

    return (
      <Modal title="Delete Query" onExit={onToggleModal}>
        <p>Are you sure you want to delete the selected queries?</p>
        <div className={`${baseClass}__modal-btn-wrap`}>
          <Button onClick={onDeleteQueries} variant="alert">
            Delete
          </Button>
          <Button onClick={onToggleModal} variant="inverse">
            Cancel
          </Button>
        </div>
      </Modal>
    );
  };

  renderSidePanel = () => {
    const { goToEditQueryPage } = this;
    const { selectedQuery } = this.props;

    if (!selectedQuery) {
      // FIXME: Render QueryDetailsSidePanel when Fritz has completed the mock
      return (
        <SecondarySidePanelContainer>
          <p className={`${baseClass}__empty-label`}>Query</p>
          <p className={`${baseClass}__empty-description`}>
            No query selected.
          </p>
        </SecondarySidePanelContainer>
      );
    }

    return (
      <QueryDetailsSidePanel
        onEditQuery={goToEditQueryPage}
        query={selectedQuery}
      />
    );
  };

  render() {
    const { checkedQueryIDs, queriesFilter } = this.state;
    const {
      getQueries,
      onCheckAllQueries,
      onCheckQuery,
      onSelectQuery,
      onDblClickQuery,
      onFilterQueries,
      renderCTAs,
      renderModal,
      renderSidePanel,
    } = this;
    const { loadingQueries, queries: allQueries, selectedQuery } = this.props;
    const queries = getQueries();
    const queriesCount = queries.length;
    const queriesTotalDisplay =
      queriesCount === 1 ? "1 query" : `${queriesCount} queries`;
    const isQueriesAvailable = allQueries.length > 0;

    if (loadingQueries) {
      return false;
    }

    return (
      <div className={`${baseClass} has-sidebar`}>
        <div className={`${baseClass}__wrapper body-wrap`}>
          <div className={`${baseClass}__header-wrap`}>
            <h1 className={`${baseClass}__title`}>Queries</h1>
            {renderCTAs()}
          </div>
          <div className={`${baseClass}__filter-and-cta`}>
            <div className={`${baseClass}__filter-queries`}>
              <InputField
                name="query-filter"
                onChange={onFilterQueries}
                placeholder="Filter queries"
                value={queriesFilter}
              />
              <KolideIcon name="search" />
            </div>
          </div>
          <p className={`${baseClass}__query-count`}>{queriesTotalDisplay}</p>
          <QueriesList
            checkedQueryIDs={checkedQueryIDs}
            isQueriesAvailable={isQueriesAvailable}
            onCheckAll={onCheckAllQueries}
            onCheckQuery={onCheckQuery}
            onSelectQuery={onSelectQuery}
            onDblClickQuery={onDblClickQuery}
            queries={queries}
            selectedQuery={selectedQuery}
          />
        </div>
        {renderSidePanel()}
        {renderModal()}
      </div>
    );
  }
}

const mapStateToProps = (state, { location }) => {
  const queryEntities = entityGetter(state).get("queries");
  const { entities: queries } = queryEntities;
  const selectedQueryID = get(location, "query.selectedQuery");
  const selectedQuery =
    selectedQueryID && queryEntities.findBy({ id: selectedQueryID });
  const { loading: loadingQueries } = state.entities.queries;

  return { loadingQueries, queries, selectedQuery };
};

export default connect(mapStateToProps)(ManageQueriesPage);
