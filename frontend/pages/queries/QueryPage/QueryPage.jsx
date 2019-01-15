import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';
import { connect } from 'react-redux';
import FileSaver from 'file-saver';
import { clone, filter, includes, isEqual, merge } from 'lodash';
import moment from 'moment';
import { push } from 'react-router-redux';

import Kolide from 'kolide';
import campaignHelpers from 'redux/nodes/entities/campaigns/helpers';
import convertToCSV from 'utilities/convert_to_csv';
import debounce from 'utilities/debounce';
import deepDifference from 'utilities/deep_difference';
import entityGetter from 'redux/utilities/entityGetter';
import { formatSelectedTargetsForApi } from 'kolide/helpers';
import helpers from 'pages/queries/QueryPage/helpers';
import hostActions from 'redux/nodes/entities/hosts/actions';
import hostInterface from 'interfaces/host';
import QueryForm from 'components/forms/queries/QueryForm';
import osqueryTableInterface from 'interfaces/osquery_table';
import queryActions from 'redux/nodes/entities/queries/actions';
import queryInterface from 'interfaces/query';
import QueryPageSelectTargets from 'components/queries/QueryPageSelectTargets';
import QueryResultsTable from 'components/queries/QueryResultsTable';
import QuerySidePanel from 'components/side_panels/QuerySidePanel';
import { renderFlash } from 'redux/nodes/notifications/actions';
import { toggleSmallNav } from 'redux/nodes/app/actions';
import { selectOsqueryTable, setSelectedTargets, setSelectedTargetsQuery } from 'redux/nodes/components/QueryPages/actions';
import targetInterface from 'interfaces/target';
import validateQuery from 'components/forms/validators/validate_query';

const baseClass = 'query-page';
const DEFAULT_CAMPAIGN = {
  hosts_count: {
    total: 0,
  },
};

const QUERY_RESULTS_OPTIONS = {
  FULL_SCREEN: 'FULL_SCREEN',
  SHRINKING: 'SHRINKING',
};

export class QueryPage extends Component {
  static propTypes = {
    dispatch: PropTypes.func,
    errors: PropTypes.shape({
      base: PropTypes.string,
    }),
    hostIDs: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    hostUUIDs: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    isSmallNav: PropTypes.bool.isRequired,
    loadingQueries: PropTypes.bool.isRequired,
    location: PropTypes.shape({
      pathname: PropTypes.string,
    }),
    query: queryInterface,
    selectedHosts: PropTypes.arrayOf(hostInterface),
    selectedOsqueryTable: osqueryTableInterface,
    selectedTargets: PropTypes.arrayOf(targetInterface),
    title: PropTypes.string,
  };

  static defaultProps = {
    loadingQueries: false,
    selectedHosts: [],
  };

  constructor (props) {
    super(props);

    this.state = {
      campaign: DEFAULT_CAMPAIGN,
      queryIsRunning: false,
      queryText: props.query.query,
      runQueryMilliseconds: 0,
      targetsCount: 0,
      targetsError: null,
      queryResultsToggle: null,
      queryPosition: {},
    };

    this.csvQueryName = 'Query Results';
  }

  componentWillMount () {
    const { dispatch, hostIDs, hostUUIDs, selectedHosts, selectedTargets } = this.props;

    if (((hostIDs && hostIDs.length) || (hostUUIDs && hostUUIDs.length)) > 0) {
      dispatch(hostActions.loadAll());
    }

    helpers.selectHosts(dispatch, {
      hosts: selectedHosts,
      selectedTargets,
    });

    return false;
  }

  componentWillReceiveProps (nextProps) {
    const { dispatch, location, selectedHosts, selectedTargets } = nextProps;
    const nextPathname = location.pathname;
    const { pathname } = this.props.location;

    if (nextPathname !== pathname) {
      this.resetCampaignAndTargets();
    }

    if (!isEqual(selectedHosts, this.props.selectedHosts)) {
      helpers.selectHosts(dispatch, {
        hosts: selectedHosts,
        selectedTargets,
      });
    }

    return false;
  }

  componentWillUnmount () {
    const { dispatch, isSmallNav } = this.props;
    const { document: { body } } = global;

    this.resetCampaignAndTargets();

    if (isSmallNav) {
      dispatch(toggleSmallNav);
    }

    if (this.runQueryInterval) {
      clearInterval(this.runQueryInterval);
    }

    body.style.overflow = 'visible';

    return false;
  }

  onChangeQueryFormField = (fieldName, value) => {
    if (fieldName === 'name') {
      this.csvQueryName = value;
    }

    if (fieldName === 'query') {
      this.setState({ queryText: value });
    }

    return false;
  }

  onExportQueryResults = (evt) => {
    evt.preventDefault();

    const { campaign } = this.state;
    const { query_results: queryResults } = campaign;

    if (queryResults) {
      const csv = convertToCSV(queryResults, (fields) => {
        const result = filter(fields, f => f !== 'host_hostname');

        result.unshift('host_hostname');

        return result;
      });
      const formattedTime = moment(new Date()).format('MM-DD-YY hh-mm-ss');
      const filename = `${this.csvQueryName} (${formattedTime}).csv`;
      const file = new global.window.File([csv], filename, { type: 'text/csv' });

      FileSaver.saveAs(file);
    }

    return false;
  }

  onFetchTargets = (query, targetResponse) => {
    const { dispatch } = this.props;
    const {
      targets_count: targetsCount,
    } = targetResponse;

    dispatch(setSelectedTargetsQuery(query));
    this.setState({ targetsCount });

    return false;
  }

  onOsqueryTableSelect = (tableName) => {
    const { dispatch } = this.props;

    dispatch(selectOsqueryTable(tableName));

    return false;
  }

  onRunQuery = debounce(() => {
    const { queryText } = this.state;
    const { dispatch, selectedTargets } = this.props;
    const { error } = validateQuery(queryText);

    if (!selectedTargets.length) {
      this.setState({ targetsError: 'You must select at least one target to run a query' });

      return false;
    }

    if (error) {
      dispatch(renderFlash('error', error));

      return false;
    }

    const { destroyCampaign, removeSocket } = this;
    const selected = formatSelectedTargetsForApi(selectedTargets);

    removeSocket();
    destroyCampaign();

    Kolide.queries.run({ query: queryText, selected })
      .then((campaignResponse) => {
        return Kolide.websockets.queries.run(campaignResponse.id)
          .then((socket) => {
            this.setupDistributedQuery(socket);

            this.setState({
              campaign: campaignResponse,
              queryIsRunning: true,
            });

            this.socket.onmessage = ({ data }) => {
              const socketData = JSON.parse(data);
              const { previousSocketData } = this;

              if (previousSocketData && isEqual(socketData, previousSocketData)) {
                return false;
              }
              this.previousSocketData = socketData;

              this.setState(campaignHelpers.updateCampaignState(socketData));

              if (socketData.type === 'status' && socketData.data.status === 'finished') {
                return this.teardownDistributedQuery();
              }

              return false;
            };
          });
      })
      .catch((campaignError) => {
        if (campaignError === 'resource already created') {
          dispatch(renderFlash('error', 'A campaign with the provided query text has already been created'));

          return false;
        }

        dispatch(renderFlash('error', campaignError));

        return false;
      });

    return false;
  })

  onSaveQueryFormSubmit = debounce((formData) => {
    const { dispatch } = this.props;
    const { error } = validateQuery(formData.query);

    if (error) {
      dispatch(renderFlash('error', error));

      return false;
    }

    return dispatch(queryActions.create(formData))
      .then((query) => {
        dispatch(push(`/queries/${query.id}`));
      })
      .catch(() => false);
  })

  onStopQuery = (evt) => {
    evt.preventDefault();

    const { teardownDistributedQuery } = this;

    return teardownDistributedQuery();
  }

  onTargetSelect = (selectedTargets) => {
    const { dispatch } = this.props;

    this.setState({ targetsError: null });

    dispatch(setSelectedTargets(selectedTargets));

    return false;
  }

  onUpdateQuery = (formData) => {
    const { dispatch, query } = this.props;
    const updatedQuery = deepDifference(formData, query);

    dispatch(queryActions.update(query, updatedQuery))
      .then(() => {
        dispatch(renderFlash('success', 'Query updated!'));
      });

    return false;
  };

  onToggleQueryFullScreen = (evt) => {
    const { document: { body }, window } = global;
    const { queryResultsToggle, queryPosition } = this.state;
    const { dispatch } = this.props;
    window.scrollTo(0, 0);
    const { parentNode: { parentNode: parent } } = evt.currentTarget;
    const { parentNode: grandParent } = parent;
    const rect = parent.getBoundingClientRect();

    const defaultPosition = {
      top: `${rect.top}px`,
      left: `${rect.left}px`,
      right: `${rect.right - rect.left}px`,
      bottom: `${rect.bottom - rect.top}px`,
      maxWidth: `${parent.offsetWidth}px`,
      maxHeight: `${parent.offsetHeight}px`,
      position: 'fixed',
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
        body.style.overflow = 'hidden';
        dispatch(toggleSmallNav);
        merge(parent.style, newPosition);
        grandParent.style.height = `${newPosition.maxHeight}`;
      };
    } else {
      newState = {
        queryResultsToggle: QUERY_RESULTS_OPTIONS.SHRINKING,
      };

      callback = () => {
        body.style.overflow = 'visible';
        dispatch(toggleSmallNav);
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
  }

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
  }

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
  }

  destroyCampaign = () => {
    const { campaign } = this.state;

    if (this.campaign || campaign) {
      this.campaign = null;
      this.setState({ campaign: DEFAULT_CAMPAIGN });
    }

    return false;
  }

  removeSocket = () => {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
      this.previousSocketData = null;
    }

    return false;
  }

  resetCampaignAndTargets = () => {
    const { destroyCampaign, removeSocket } = this;
    const { dispatch } = this.props;

    destroyCampaign();
    dispatch(setSelectedTargets([]));
    removeSocket();

    return false;
  }

  renderResultsTable = () => {
    const {
      campaign,
      queryIsRunning,
      queryResultsToggle,
      runQueryMilliseconds,
    } = this.state;
    const { onExportQueryResults, onToggleQueryFullScreen, onRunQuery, onStopQuery, onTargetSelect } = this;
    const loading = queryIsRunning && !campaign.hosts_count.total;
    const isQueryFullScreen = queryResultsToggle === QUERY_RESULTS_OPTIONS.FULL_SCREEN;
    const isQueryShrinking = queryResultsToggle === QUERY_RESULTS_OPTIONS.SHRINKING;
    const resultsClasses = classnames(`${baseClass}__results`, 'body-wrap', {
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
  }

  renderTargetsInput = () => {
    const { onFetchTargets, onRunQuery, onStopQuery, onTargetSelect } = this;
    const { campaign, queryIsRunning, targetsCount, targetsError, runQueryMilliseconds } = this.state;
    const { selectedTargets } = this.props;

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
      />
    );
  }

  render () {
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
    } = this;
    const { queryIsRunning } = this.state;
    const {
      errors,
      loadingQueries,
      query,
      selectedOsqueryTable,
      title,
    } = this.props;

    if (loadingQueries) {
      return false;
    }

    return (
      <div className={`${baseClass} has-sidebar`}>
        <div className={`${baseClass}__content`}>
          <div className={`${baseClass}__form body-wrap`}>
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
            />
          </div>
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
  const { id: queryID } = ownProps.params;
  const reduxQuery = entityGetter(state).get('queries').findBy({ id: queryID });
  const { queryText, selectedOsqueryTable } = state.components.QueryPages;
  const { errors, loading: loadingQueries } = state.entities.queries;
  const queryStub = { description: '', name: '', query: queryText };
  const query = reduxQuery || queryStub;
  const { selectedTargets } = state.components.QueryPages;
  const { host_ids: hostIDs, host_uuids: hostUUIDs } = ownProps.location.query;
  const { isSmallNav } = state.app;
  const title = queryID ? 'Edit Query' : 'New Query';
  let selectedHosts = [];

  if (!queryID && ((hostIDs && hostIDs.length) || (hostUUIDs && hostUUIDs.length)) > 0) {
    const hostIDsArr = Array.isArray(hostIDs) ? hostIDs : [hostIDs];
    const hostUUIDsArr = Array.isArray(hostUUIDs) ? hostUUIDs : [hostUUIDs];
    const { entities: hosts } = stateEntities.get('hosts');
    // hostIDs are URL params so they are strings and comparison with ints may
    // need conversion.
    const hostFilter = h => includes(hostIDsArr, String(h.id)) || includes(hostUUIDsArr, String(h.uuid));
    selectedHosts = filter(hosts, hostFilter);
  }

  return {
    errors,
    hostIDs,
    hostUUIDs,
    isSmallNav,
    loadingQueries,
    query,
    selectedOsqueryTable,
    selectedHosts,
    selectedTargets,
    title,
  };
};

export default connect(mapStateToProps)(QueryPage);
