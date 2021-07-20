import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { filter, get, includes, pull } from "lodash";
import { push } from "react-router-redux";

import permissionUtils from "utilities/permissions";
import simpleSearch from "utilities/simple_search";
import entityGetter from "redux/utilities/entityGetter";
import Button from "components/buttons/Button";
import Modal from "components/modals/Modal";
import SecondarySidePanelContainer from "components/side_panels/SecondarySidePanelContainer";
import QueryDetailsSidePanel from "components/side_panels/QueryDetailsSidePanel";
import TableContainer from "components/TableContainer";

import PATHS from "router/paths";
import queryActions from "redux/nodes/entities/queries/actions";
import queryInterface from "interfaces/query";
import userInterface from "interfaces/user";
import { renderFlash } from "redux/nodes/notifications/actions";

import {
  generateTableHeaders,
  generateTableData,
} from "./ManageQueriesTableConfig";

import DeleteIcon from "../../../../assets/images/icon-action-delete-14x14@2x.png";

const baseClass = "manage-queries-page";

export class ManageQueriesPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    loadingQueries: PropTypes.bool.isRequired,
    queries: PropTypes.arrayOf(queryInterface),
    selectedQuery: queryInterface,
    currentUser: userInterface,
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

    this.tableHeaders = generateTableHeaders(
      permissionUtils.isOnlyObserver(props.currentUser)
    );

    this.secondarySelectActions = [
      {
        callback: (selectedRows) => console.log("clicked delete: ", selectedRows),
        name: "delete",
      },
      {
        callback: (selectedRows) => console.log("clicked edit: ", selectedRows),
        name: "edit",
      },
    ];
  }

  componentWillMount() {
    const { dispatch } = this.props;

    dispatch(queryActions.loadAll()).catch(() => false);

    return false;
  }

  // componentWillReceiveProps(nextProps) {
  //   this.setState({ queryState: nextProps.queries });
  // }

  onDeleteQueries = (selectedQueryIds) => {
    const { dispatch } = this.props;
    const { destroy } = queryActions;

    const promises = selectedQueryIds.map((queryID) => {
      return dispatch(destroy({ id: queryID }));
    });

    return Promise.all(promises)
      .then(() => {
        dispatch(renderFlash("success", "Queries successfully deleted."));

        this.setState({ showModal: false });

        return false;
      })
      .catch(() => {
        dispatch(renderFlash("error", "Something went wrong."));

        this.setState({ showModal: false });

        return false;
      });
  };

  // onCheckAllQueries = (shouldCheck) => {
  //   if (shouldCheck) {
  //     const queries = this.getQueries();
  //     const checkedQueryIDs = queries.map((query) => query.id);

  //     this.setState({ allQueriesChecked: true, checkedQueryIDs });

  //     return false;
  //   }

  //   this.setState({ allQueriesChecked: false, checkedQueryIDs: [] });

  //   return false;
  // };

  // onCheckQuery = (checked, id) => {
  //   const { checkedQueryIDs } = this.state;
  //   const newCheckedQueryIDs = checked
  //     ? checkedQueryIDs.concat(id)
  //     : pull(checkedQueryIDs, id);

  //   this.setState({
  //     allQueriesChecked: false,
  //     checkedQueryIDs: newCheckedQueryIDs,
  //   });

  //   return false;
  // };

  // onFilterQueries = (queriesFilter) => {
  //   this.setState({ queriesFilter });

  //   return false;
  // };

  // onSelectQuery = (selectedQuery) => {
  //   const { dispatch } = this.props;
  //   const locationObject = {
  //     pathname: PATHS.MANAGE_QUERIES,
  //     query: { selectedQuery: selectedQuery.id },
  //   };

  //   dispatch(push(locationObject));

  //   return false;
  // };

  // onDblClickQuery = (selectedQuery) => {
  //   const { dispatch } = this.props;

  //   dispatch(push(PATHS.EDIT_QUERY(selectedQuery)));

  //   return false;
  // };

  onTableQueryChange = (queryData) => {
    const { queries } = this.props;
    const {
      // pageIndex,
      // pageSize,
      searchQuery,
      // sortHeader,
      // sortDirection,
    } = queryData;
    // let sortBy = [];
    // if (sortHeader !== "") {
    //   sortBy = [{ id: sortHeader, direction: sortDirection }];
    // }

    if (!searchQuery) {
      this.setState({ queriesFilter: "" });
      return;
    }

    this.setState({ queriesFilter: searchQuery });
  };

  onToggleModal = () => {
    const { showModal } = this.state;

    this.setState({ showModal: !showModal });

    return false;
  };

  onActionButtonClick = () => {
    const { goToNewQueryPage } = this;
    console.log("Clicked Action Button");
    goToNewQueryPage();
  };

  onSelectActionButtonClick = (selectedQueryIds) => {
    const { onDeleteQueries } = this;
    console.log("Clicked Select Action Button");
    onDeleteQueries(selectedQueryIds);
    // const { onDeleteQueries } = this;
    // const isOnlyObserver = permissionUtils.isOnlyObserver(
    //   this.props.currentUser
    // );

    // if (!isOnlyObserver) {
    //   onDeleteQueries(selectedQueryIds);
    // }
    // TODO render confirmation modal?
    // TODO render flash for second delete?
  };

  getQueries = () => {
    const { queriesFilter } = this.state;
    const { queries } = this.props;

    if (!queriesFilter) {
      // return queries || [];
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

  generateActionButtonText = () => {
    const { currentUser } = this.props;

    if (!permissionUtils.isOnlyObserver(currentUser)) {
      return "Create new query";
    }
    // The action button will not be rendered if TableContainer's actionButtonText prop is falsey
    return false;
  };

  generateSelectActionButtonText = () => {
    const { currentUser } = this.props;

    if (!permissionUtils.isOnlyObserver(currentUser)) {
      return "Delete";
    }
    return "Default Select Action";
  };

  // renderCTAs = () => {
  //   const { goToNewQueryPage, onToggleModal } = this;
  //   const { currentUser } = this.props;

  //   const checkedQueryCount = this.state.checkedQueryIDs.length;

  //   if (checkedQueryCount) {
  //     return (
  //       <div className={`${baseClass}__ctas`}>
  //         <span className={`${baseClass}__selected-count`}>
  //           <strong>{checkedQueryCount}</strong> selected
  //         </span>
  //         <Button onClick={onToggleModal} variant="text-icon">
  //           <>
  //             <img src={DeleteIcon} alt="Delete query icon" />
  //             Delete
  //           </>
  //         </Button>
  //       </div>
  //     );
  //   }

  //   // Render option to create new query only for maintainers and admin
  //   if (!permissionUtils.isOnlyObserver(currentUser)) {
  //     return (
  //       <Button variant="brand" onClick={goToNewQueryPage}>
  //         Create new query
  //       </Button>
  //     );
  //   }

  //   return null;
  // };

  renderModal = () => {
    const { onDeleteQueries, onToggleModal } = this;
    const { showModal } = this.state;

    if (!showModal) {
      return false;
    }

    return (
      <Modal title="Delete query" onExit={onToggleModal}>
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
    const { selectedQuery, currentUser } = this.props;

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
        currentUser={currentUser}
      />
    );
  };

  render() {
    const { checkedQueryIDs, queriesFilter, queryState } = this.state;
    const {
      onActionButtonClick,
      onSelectActionButtonClick,
      onTableQueryChange,
      getQueries,
      generateActionButtonText,
      generateSelectActionButtonText,
      // onCheckAllQueries,
      // onCheckQuery,
      // onSelectQuery,
      // onDblClickQuery,
      // onFilterQueries,
      renderModal,
      renderSidePanel,
      tableHeaders,
      secondarySelectActions,
    } = this;
    const {
      loadingQueries,
      // selectedQuery,
    } = this.props;

    // if (loadingQueries) {
    //   return false;
    // }

    return (
      <div className={`${baseClass} has-sidebar`}>
        <div className={`${baseClass}__wrapper body-wrap`}>
          <div className={`${baseClass}__header-wrap`}>
            <h1 className={`${baseClass}__title`}>Queries</h1>
          </div>
          <TableContainer
            columns={tableHeaders}
            data={generateTableData(getQueries())}
            isLoading={loadingQueries}
            defaultSortHeader={"name"}
            defaultSortDirection={"desc"}
            inputPlaceHolder={"Search"}
            actionButtonText={generateActionButtonText()}
            selectActionButtonText={generateSelectActionButtonText()}
            onActionButtonClick={onActionButtonClick}
            onSelectActionClick={onSelectActionButtonClick}
            onQueryChange={onTableQueryChange}
            resultsTitle={"queries"}
            emptyComponent={null}
            searchable
            secondarySelectActions={secondarySelectActions}
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
  const currentUser = state.auth.user;

  return { loadingQueries, queries, selectedQuery, currentUser };
};

export default connect(mapStateToProps)(ManageQueriesPage);
