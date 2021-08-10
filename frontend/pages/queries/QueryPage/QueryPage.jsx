import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import { connect } from "react-redux";
import FileSaver from "file-saver";
import { clone, filter, includes, isEqual, merge } from "lodash";
import moment from "moment";
import { push } from "react-router-redux";
import { Link } from "react-router";

import Fleet from "fleet";
import campaignHelpers from "redux/nodes/entities/campaigns/helpers";
import convertToCSV from "utilities/convert_to_csv";
import debounce from "utilities/debounce";
import deepDifference from "utilities/deep_difference";
import permissionUtils from "utilities/permissions";
import entityGetter from "redux/utilities/entityGetter";
import { formatSelectedTargetsForApi } from "fleet/helpers";
import helpers from "pages/queries/QueryPage/helpers";
import hostInterface from "interfaces/host";
import Button from "components/buttons/Button";
import FleetAce from "components/FleetAce";
import WarningBanner from "components/WarningBanner";
import QueryForm from "components/forms/queries/QueryForm";
import osqueryTableInterface from "interfaces/osquery_table";
import queryActions from "redux/nodes/entities/queries/actions";
import queryInterface from "interfaces/query";
import userInterface from "interfaces/user";
import QueryPageSelectTargets from "components/queries/QueryPageSelectTargets";
import QueryResultsTable from "components/queries/QueryResultsTable";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import { renderFlash } from "redux/nodes/notifications/actions";
import {
  selectOsqueryTable,
  setSelectedTargets,
  setSelectedTargetsQuery,
} from "redux/nodes/components/QueryPages/actions";
import targetInterface from "interfaces/target";
import validateQuery from "components/forms/validators/validate_query";
import PATHS from "router/paths";
import BackChevron from "../../../../assets/images/icon-chevron-down-9x6@2x.png";

const baseClass = "query-page";
const DEFAULT_CAMPAIGN = {
  hosts_count: {
    total: 0,
  },
};

const QUERY_RESULTS_OPTIONS = {
  FULL_SCREEN: "FULL_SCREEN",
  SHRINKING: "SHRINKING",
};

export class QueryPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    errors: PropTypes.shape({
      base: PropTypes.string,
    }),
    loadingQueries: PropTypes.bool.isRequired,
    location: PropTypes.shape({
      pathname: PropTypes.string,
    }),
    query: queryInterface,
    queryId: PropTypes.number,
    selectedHosts: PropTypes.arrayOf(hostInterface),
    selectedOsqueryTable: osqueryTableInterface,
    selectedTargets: PropTypes.arrayOf(targetInterface),
    title: PropTypes.string,
    requestHost: PropTypes.bool,
    hostId: PropTypes.string,
    currentUser: userInterface,
    isBasicTier: PropTypes.bool,
  };

  static defaultProps = {
    loadingQueries: false,
    query: { description: "", name: "", query: "SELECT * FROM osquery_info" },
    selectedHosts: [],
  };

  constructor(props) {
    super(props);

    this.state = {
      campaign: DEFAULT_CAMPAIGN,
      queryIsRunning: false,
      runQueryMilliseconds: 0,
      targetsCount: 0,
      targetsError: null,
      queryResultsToggle: null,
      queryPosition: {},
      selectRelatedHostTarget: true,
      observerShowSql: false,
    };

    this.csvQueryName = "Query Results";
  }

  componentWillMount() {
    const { dispatch, selectedHosts, selectedTargets } = this.props;

    Fleet.status.live_query().catch((response) => {
      try {
        const error = response.message.errors[0].reason;
        this.setState({ liveQueryError: error });
      } catch (e) {
        const error = `Unknown error: ${e}`;
        this.setState({ liveQueryError: error });
      }
    });

    helpers.selectHosts(dispatch, {
      hosts: selectedHosts,
      selectedTargets,
    });

    return false;
  }

  componentDidMount() {
    const { dispatch, requestHost, hostId } = this.props;

    // A fetch call is required for the host data if we do not already have the host
    // data that is related to this query.
    // e.g. coming into the app from a bookmark link /queries/new?host_ids=4
    if (requestHost) {
      const { fetchHost } = helpers;
      fetchHost(dispatch, hostId).then(() => {
        this.setState({ selectRelatedHostTarget: true });
      });
    }
  }

  componentWillReceiveProps(nextProps) {
    const { location } = nextProps;
    const nextPathname = location.pathname;
    const { pathname } = this.props.location;

    // this will initially select the related host for queries. This should only happen one time,
    // and only when a query has a related host.
    const { dispatch, selectedHosts, selectedTargets } = nextProps;
    if (this.state.selectRelatedHostTarget) {
      helpers.selectHosts(dispatch, {
        hosts: selectedHosts,
        selectedTargets,
      });
      this.setState({ selectRelatedHostTarget: false });
    }

    if (nextPathname !== pathname) {
      this.resetCampaignAndTargets();
    }

    return false;
  }

  componentWillUnmount() {
    const {
      document: { body },
    } = global;

    this.resetCampaignAndTargets();

    if (this.runQueryInterval) {
      clearInterval(this.runQueryInterval);
    }

    body.style.overflow = "visible";

    return false;
  }

  onChangeQueryFormField = (fieldName, value) => {
    if (fieldName === "name") {
      this.csvQueryName = value;
    }

    if (fieldName === "query") {
      this.setState({ queryText: value });
    }

    return false;
  };

  onExportQueryResults = (evt) => {
    evt.preventDefault();

    const { campaign } = this.state;
    const { query_results: queryResults } = campaign;

    if (queryResults) {
      const csv = convertToCSV(queryResults, (fields) => {
        const result = filter(fields, (f) => f !== "host_hostname");

        result.unshift("host_hostname");

        return result;
      });
      const formattedTime = moment(new Date()).format("MM-DD-YY hh-mm-ss");
      const filename = `${this.csvQueryName} (${formattedTime}).csv`;
      const file = new global.window.File([csv], filename, {
        type: "text/csv",
      });

      FileSaver.saveAs(file);
    }

    return false;
  };

  onExportErrorsResults = (evt) => {
    evt.preventDefault();

    const { campaign } = this.state;
    const { errors } = campaign;

    if (errors) {
      const csv = convertToCSV(errors, (fields) => {
        const result = filter(fields, (f) => f !== "host_hostname");

        result.unshift("host_hostname");

        return result;
      });
      const formattedTime = moment(new Date()).format("MM-DD-YY hh-mm-ss");
      const filename = `${this.csvQueryName} Errors (${formattedTime}).csv`;
      const file = new global.window.File([csv], filename, {
        type: "text/csv",
      });

      FileSaver.saveAs(file);
    }

    return false;
  };

  onFetchTargets = (query, targetResponse) => {
    const { dispatch } = this.props;

    const { targets_count: targetsCount } = targetResponse;

    dispatch(setSelectedTargetsQuery(query));
    this.setState({ targetsCount });

    return false;
  };

  onOsqueryTableSelect = (tableName) => {
    const { dispatch } = this.props;

    dispatch(selectOsqueryTable(tableName));

    return false;
  };

  onRunQuery = debounce(() => {
    const { queryText, targetsCount } = this.state;
    const { query } = this.props.query;
    const query_id = parseInt(this.props.queryId, 10) || null;
    const sql = queryText || query;
    const { dispatch, selectedTargets } = this.props;
    const { error } = validateQuery(sql);

    if (!selectedTargets.length) {
      this.setState({
        targetsError: "You must select at least one target to run a query",
      });

      return false;
    }

    if (!targetsCount) {
      this.setState({
        targetsError:
          "You must select a target with at least one host to run a query",
      });

      return false;
    }

    if (error) {
      dispatch(renderFlash("error", error));

      return false;
    }

    const { destroyCampaign, removeSocket } = this;
    const selected = formatSelectedTargetsForApi(selectedTargets);

    removeSocket();
    destroyCampaign();

    Fleet.queries
      .run({ query: sql, selected, query_id })
      .then((campaignResponse) => {
        return Fleet.websockets.queries
          .run(campaignResponse.id)
          .then((socket) => {
            this.setupDistributedQuery(socket);

            this.setState({
              campaign: campaignResponse,
              queryIsRunning: true,
            });

            this.socket.onmessage = ({ data }) => {
              const socketData = JSON.parse(data);
              const { previousSocketData } = this;

              if (
                previousSocketData &&
                isEqual(socketData, previousSocketData)
              ) {
                return false;
              }
              this.previousSocketData = socketData;

              this.setState(campaignHelpers.updateCampaignState(socketData));

              if (
                socketData.type === "status" &&
                socketData.data.status === "finished"
              ) {
                return this.teardownDistributedQuery();
              }

              return false;
            };
          });
      })
      .catch((campaignError) => {
        console.log(campaignError);
        // TODO Revisit after taking a deeper look at error handling related to the Fleet.entities
        // and flash_messages components in light of issues with those in other instances,
        // especially as it concerns async errors.

        dispatch(push("/500"));

        return false;
      });

    return false;
  });

  onSaveQueryFormSubmit = debounce((formData) => {
    const { dispatch } = this.props;
    const { error } = validateQuery(formData.query);

    if (error) {
      dispatch(renderFlash("error", error));

      return false;
    }

    return dispatch(queryActions.create(formData))
      .then((query) => {
        dispatch(push(PATHS.EDIT_QUERY(query)));
        dispatch(renderFlash("success", "Query created!"));
      })
      .catch(() => false);
  });

  onStopQuery = (evt) => {
    evt.preventDefault();

    const { teardownDistributedQuery } = this;

    return teardownDistributedQuery();
  };

  onTargetSelect = (selectedTargets) => {
    const { dispatch } = this.props;

    this.setState({ targetsError: null });

    dispatch(setSelectedTargets(selectedTargets));

    return false;
  };

  onUpdateQuery = (formData) => {
    const { dispatch, query } = this.props;
    const updatedQuery = deepDifference(formData, query);

    dispatch(queryActions.update(query, updatedQuery)).then(() => {
      dispatch(renderFlash("success", "Query updated!"));
    });

    return false;
  };

  onToggleQueryFullScreen = (evt) => {
    const {
      document: { body },
      window,
    } = global;
    const { queryResultsToggle, queryPosition } = this.state;
    const {
      parentNode: { parentNode: parent },
    } = evt.currentTarget;
    const { parentNode: grandParent } = parent;
    const rect = parent.getBoundingClientRect();

    const defaultPosition = {
      top: `${rect.top}px`,
      left: `${rect.left}px`,
      right: `${rect.right - rect.left}px`,
      bottom: `${rect.bottom - rect.top}px`,
      maxWidth: `${parent.offsetWidth}px`,
      maxHeight: `${parent.offsetHeight}px`,
      position: "fixed",
    };

    const resetPosition = {
      position: null,
      maxWidth: null,
      minWidth: null,
      maxHeight: null,
      minHeight: null,
      top: null,
      right: null,
      bottom: null,
      left: null,
    };

    let newPosition = clone(defaultPosition);
    let newState;
    let callback;

    if (queryResultsToggle !== QUERY_RESULTS_OPTIONS.FULL_SCREEN) {
      newState = {
        queryResultsToggle: QUERY_RESULTS_OPTIONS.FULL_SCREEN,
        queryPosition: defaultPosition,
      };

      callback = () => {
        body.style.overflow = "hidden";
        merge(parent.style, newPosition);
        grandParent.style.height = `${newPosition.maxHeight}`;
      };
    } else {
      newState = {
        queryResultsToggle: QUERY_RESULTS_OPTIONS.SHRINKING,
      };

      callback = () => {
        body.style.overflow = "visible";
        newPosition = queryPosition;
        merge(parent.style, newPosition);
        grandParent.style.height = `${newPosition.maxHeight}`;

        window.setTimeout(() => {
          merge(parent.style, resetPosition);
          this.setState({ queryResultsToggle: null });
        }, 500);
      };
    }

    this.setState(newState, callback);

    return false;
  };

  setupDistributedQuery = (socket) => {
    this.socket = socket;
    const update = () => {
      const { runQueryMilliseconds } = this.state;

      this.setState({ runQueryMilliseconds: runQueryMilliseconds + 1000 });
    };

    if (!this.runQueryInterval) {
      this.runQueryInterval = setInterval(update, 1000);
    }

    return false;
  };

  teardownDistributedQuery = () => {
    const { runQueryInterval } = this;

    if (runQueryInterval) {
      clearInterval(runQueryInterval);
      this.runQueryInterval = null;
    }

    this.setState({
      queryIsRunning: false,
      runQueryMilliseconds: 0,
    });
    this.removeSocket();

    return false;
  };

  destroyCampaign = () => {
    const { campaign } = this.state;

    if (this.campaign || campaign) {
      this.campaign = null;
      this.setState({ campaign: DEFAULT_CAMPAIGN });
    }

    return false;
  };

  removeSocket = () => {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
      this.previousSocketData = null;
    }

    return false;
  };

  resetCampaignAndTargets = () => {
    const { destroyCampaign, removeSocket } = this;
    const { dispatch } = this.props;

    destroyCampaign();
    dispatch(setSelectedTargets([]));
    removeSocket();

    return false;
  };

  renderLiveQueryWarning = () => {
    const { liveQueryError } = this.state;

    if (!liveQueryError) {
      return false;
    }

    return (
      <WarningBanner className={`${baseClass}__warning`} shouldShowWarning>
        <h2 className={`${baseClass}__warning-title`}>
          Live query request failed
        </h2>
        <p>
          <span>Error:</span> {liveQueryError}
        </p>
      </WarningBanner>
    );
  };

  renderResultsTable = () => {
    const {
      campaign,
      queryIsRunning,
      queryResultsToggle,
      runQueryMilliseconds,
    } = this.state;
    const {
      onExportQueryResults,
      onExportErrorsResults,
      onToggleQueryFullScreen,
      onRunQuery,
      onStopQuery,
      onTargetSelect,
    } = this;
    const loading = queryIsRunning && !campaign.hosts_count.total;
    const isQueryFullScreen =
      queryResultsToggle === QUERY_RESULTS_OPTIONS.FULL_SCREEN;
    const isQueryShrinking =
      queryResultsToggle === QUERY_RESULTS_OPTIONS.SHRINKING;
    const resultsClasses = classnames(`${baseClass}__results`, "body-wrap", {
      [`${baseClass}__results--loading`]: loading,
      [`${baseClass}__results--full-screen`]: isQueryFullScreen,
    });

    if (isEqual(campaign, DEFAULT_CAMPAIGN)) {
      return false;
    }

    return (
      <div className={resultsClasses}>
        <QueryResultsTable
          campaign={campaign}
          onExportQueryResults={onExportQueryResults}
          onExportErrorsResults={onExportErrorsResults}
          isQueryFullScreen={isQueryFullScreen}
          isQueryShrinking={isQueryShrinking}
          onToggleQueryFullScreen={onToggleQueryFullScreen}
          onRunQuery={onRunQuery}
          onStopQuery={onStopQuery}
          onTargetSelect={onTargetSelect}
          queryIsRunning={queryIsRunning}
          queryTimerMilliseconds={runQueryMilliseconds}
        />
      </div>
    );
  };

  renderTargetsInput = () => {
    const { onFetchTargets, onRunQuery, onStopQuery, onTargetSelect } = this;
    const {
      campaign,
      queryIsRunning,
      targetsCount,
      targetsError,
      runQueryMilliseconds,
      liveQueryError,
    } = this.state;
    const { selectedTargets, isBasicTier } = this.props;
    const queryId = this.props.query.id;

    return (
      <QueryPageSelectTargets
        campaign={campaign}
        error={targetsError}
        onFetchTargets={onFetchTargets}
        onRunQuery={onRunQuery}
        onStopQuery={onStopQuery}
        onTargetSelect={onTargetSelect}
        queryIsRunning={queryIsRunning}
        selectedTargets={selectedTargets}
        targetsCount={targetsCount}
        queryTimerMilliseconds={runQueryMilliseconds}
        disableRun={liveQueryError !== undefined}
        queryId={queryId}
        isBasicTier={isBasicTier}
      />
    );
  };

  render() {
    const {
      onChangeQueryFormField,
      onOsqueryTableSelect,
      onRunQuery,
      onSaveQueryFormSubmit,
      onStopQuery,
      onTextEditorInputChange,
      onUpdateQuery,
      renderResultsTable,
      renderTargetsInput,
      renderLiveQueryWarning,
    } = this;
    const { queryIsRunning } = this.state;
    const {
      errors,
      loadingQueries,
      query,
      selectedOsqueryTable,
      title,
      currentUser,
    } = this.props;
    const { hasSavePermissions, showDropdown } = helpers;

    const queryId = this.props.query.id;

    if (loadingQueries) {
      return false;
    }

    const QuerySql = () => (
      <div id="results" className="search-results">
        <FleetAce
          fontSize={12}
          name="query-details"
          readOnly
          showGutter
          value={query.query}
          wrapperClassName={`${baseClass}__query-preview`}
          wrapEnabled
        />
      </div>
    );

    // Shows and hides SQL for Restricted UI
    const editDisabledSql = () => {
      const toggleSql = () =>
        this.setState((prevState) => ({
          observerShowSql: !prevState.observerShowSql,
        }));
      return (
        <div>
          <Button variant="unstyled" className="sql-button" onClick={toggleSql}>
            {this.state.observerShowSql ? "Hide SQL" : "Show SQL"}
          </Button>
          {this.state.observerShowSql ? <QuerySql /> : null}
        </div>
      );
    };

    // Team maintainer: Create and run new query, but not save
    if (permissionUtils.isAnyTeamMaintainer(currentUser)) {
      // Team maintainer: Existing query
      if (queryId) {
        return (
          <div className={`${baseClass}__content`}>
            <div className={`${baseClass}__observer-query-view body-wrap`}>
              <div className={`${baseClass}__observer-query-details`}>
                <Link
                  to={PATHS.MANAGE_QUERIES}
                  className={`${baseClass}__back-link`}
                >
                  <img src={BackChevron} alt="back chevron" id="back-chevron" />
                  <span>Back to queries</span>
                </Link>
                <h1>{query.name}</h1>
                <p>{query.description}</p>
                {editDisabledSql()}
              </div>
              {renderLiveQueryWarning()}
              {renderTargetsInput()}
              {renderResultsTable()}
            </div>
          </div>
        );
      }

      // Team maintainer: New query
      return (
        <div className={`${baseClass} has-sidebar`}>
          <div className={`${baseClass}__content`}>
            <div className={`${baseClass}__form body-wrap`}>
              <Link
                to={PATHS.MANAGE_QUERIES}
                className={`${baseClass}__back-link`}
              >
                <img src={BackChevron} alt="back chevron" id="back-chevron" />
                <span>Back to queries</span>
              </Link>
              <QueryForm
                formData={query}
                handleSubmit={onSaveQueryFormSubmit}
                onChangeFunc={onChangeQueryFormField}
                onOsqueryTableSelect={onOsqueryTableSelect}
                onRunQuery={onRunQuery}
                onStopQuery={onStopQuery}
                onUpdate={onUpdateQuery}
                queryIsRunning={queryIsRunning}
                serverErrors={errors}
                selectedOsqueryTable={selectedOsqueryTable}
                title={title}
                hasSavePermissions={hasSavePermissions(currentUser)}
              />
            </div>
            {renderLiveQueryWarning()}
            {renderTargetsInput()}
            {renderResultsTable()}
          </div>
          <QuerySidePanel
            onOsqueryTableSelect={onOsqueryTableSelect}
            onTextEditorInputChange={onTextEditorInputChange}
            selectedOsqueryTable={selectedOsqueryTable}
          />
        </div>
      );
    }

    // Global Observer or Team Maintainer or Team Observer: Restricted UI
    if (
      permissionUtils.isGlobalObserver(currentUser) ||
      !permissionUtils.isOnGlobalTeam(currentUser)
    ) {
      return (
        <div className={`${baseClass}__content`}>
          <div className={`${baseClass}__observer-query-view body-wrap`}>
            <div className={`${baseClass}__observer-query-details`}>
              <Link
                to={PATHS.MANAGE_QUERIES}
                className={`${baseClass}__back-link`}
              >
                <img src={BackChevron} alt="back chevron" id="back-chevron" />
                <span>Back to queries</span>
              </Link>
              <h1>{query.name}</h1>
              <p>{query.description}</p>
              {editDisabledSql()}
            </div>
            {showDropdown(query, currentUser) && (
              <div>
                {renderLiveQueryWarning()}
                {renderTargetsInput()}
                {renderResultsTable()}
              </div>
            )}
          </div>
        </div>
      );
    }

    // Global Admin or Global Maintainer: Full functionality
    return (
      <div className={`${baseClass} has-sidebar`}>
        <div className={`${baseClass}__content`}>
          <div className={`${baseClass}__form body-wrap`}>
            <Link
              to={PATHS.MANAGE_QUERIES}
              className={`${baseClass}__back-link`}
            >
              <img src={BackChevron} alt="back chevron" id="back-chevron" />
              <span>Back to queries</span>
            </Link>
            <QueryForm
              formData={query}
              handleSubmit={onSaveQueryFormSubmit}
              onChangeFunc={onChangeQueryFormField}
              onOsqueryTableSelect={onOsqueryTableSelect}
              onRunQuery={onRunQuery}
              onStopQuery={onStopQuery}
              onUpdate={onUpdateQuery}
              queryIsRunning={queryIsRunning}
              serverErrors={errors}
              selectedOsqueryTable={selectedOsqueryTable}
              title={title}
              hasSavePermissions={hasSavePermissions(currentUser)}
            />
          </div>
          {renderLiveQueryWarning()}
          {renderTargetsInput()}
          {renderResultsTable()}
        </div>
        <QuerySidePanel
          onOsqueryTableSelect={onOsqueryTableSelect}
          onTextEditorInputChange={onTextEditorInputChange}
          selectedOsqueryTable={selectedOsqueryTable}
        />
      </div>
    );
  }
}

const mapStateToProps = (state, ownProps) => {
  const stateEntities = entityGetter(state);
  const { id: queryId } = ownProps.params;
  const query = entityGetter(state).get("queries").findBy({ id: queryId });
  const { selectedOsqueryTable } = state.components.QueryPages;
  const { errors, loading: loadingQueries } = state.entities.queries;
  const { selectedTargets } = state.components.QueryPages;
  const { host_ids: hostIDs, host_uuids: hostUUIDs } = ownProps.location.query;
  const title = queryId ? "Edit & run query" : "Custom query";
  let selectedHosts = [];

  if (((hostIDs && hostIDs.length) || (hostUUIDs && hostUUIDs.length)) > 0) {
    const hostIDsArr = Array.isArray(hostIDs) ? hostIDs : [hostIDs];
    const hostUUIDsArr = Array.isArray(hostUUIDs) ? hostUUIDs : [hostUUIDs];
    const { entities: hosts } = stateEntities.get("hosts");
    // hostIDs are URL params so they are strings and comparison with ints may
    // need conversion.
    const hostFilter = (h) =>
      includes(hostIDsArr, String(h.id)) ||
      includes(hostUUIDsArr, String(h.uuid));
    selectedHosts = filter(hosts, hostFilter);
  }

  const hostId = ownProps.location.query.host_ids;
  const relatedHost = stateEntities
    .get("hosts")
    .findBy({ id: parseInt(hostId, 10) });
  const requestHost = hostId !== undefined && relatedHost === undefined;
  const currentUser = state.auth.user;
  const config = state.app.config;

  const isBasicTier = permissionUtils.isBasicTier(config);

  return {
    errors,
    loadingQueries,
    query,
    queryId,
    selectedOsqueryTable,
    selectedHosts,
    selectedTargets,
    requestHost,
    hostId,
    title,
    currentUser,
    isBasicTier,
  };
};

export default connect(mapStateToProps)(QueryPage);
