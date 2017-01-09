import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { push } from 'react-router-redux';
import { first, isEqual, values } from 'lodash';

import Kolide from 'kolide';
import campaignActions from 'redux/nodes/entities/campaigns/actions';
import campaignInterface from 'interfaces/campaign';
import debounce from 'utilities/debounce';
import entityGetter from 'redux/utilities/entityGetter';
import { formatSelectedTargetsForApi } from 'kolide/helpers';
import QueryForm from 'components/forms/queries/QueryForm';
import osqueryTableInterface from 'interfaces/osquery_table';
import queryActions from 'redux/nodes/entities/queries/actions';
import queryInterface from 'interfaces/query';
import QueryResultsTable from 'components/queries/QueryResultsTable';
import QuerySidePanel from 'components/side_panels/QuerySidePanel';
import { renderFlash } from 'redux/nodes/notifications/actions';
import { selectOsqueryTable, setSelectedTargets, setSelectedTargetsQuery } from 'redux/nodes/components/QueryPages/actions';
import targetInterface from 'interfaces/target';
import validateQuery from 'components/forms/validators/validate_query';

class QueryPage extends Component {
  static propTypes = {
    campaign: campaignInterface,
    dispatch: PropTypes.func,
    errors: PropTypes.shape({
      base: PropTypes.string,
    }),
    query: queryInterface,
    selectedOsqueryTable: osqueryTableInterface,
    selectedTargets: PropTypes.arrayOf(targetInterface),
  };

  constructor (props) {
    super(props);

    this.state = {
      queryIsRunning: false,
      targetsCount: 0,
    };
  }

  componentWillUnmount () {
    const { destroyCampaign, removeSocket } = this;

    removeSocket();
    destroyCampaign();

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

  onRunQuery = debounce((queryText) => {
    const { dispatch, selectedTargets } = this.props;
    const { error } = validateQuery(queryText);

    if (error) {
      dispatch(renderFlash('error', error));

      return false;
    }

    const { create, update } = campaignActions;
    const { destroyCampaign, removeSocket } = this;
    const selected = formatSelectedTargetsForApi(selectedTargets);

    removeSocket();
    destroyCampaign();

    dispatch(create({ query: queryText, selected }))
      .then((campaignResponse) => {
        return Kolide.runQueryWebsocket(campaignResponse.id)
          .then((socket) => {
            this.campaign = campaignResponse;
            this.socket = socket;
            this.setState({ queryIsRunning: true });

            this.socket.onmessage = ({ data }) => {
              const socketData = JSON.parse(data);
              const { previousSocketData } = this;

              if (previousSocketData && isEqual(socketData, previousSocketData)) {
                this.previousSocketData = socketData;

                return false;
              }

              return dispatch(update(this.campaign, socketData))
                .then((updatedCampaign) => {
                  this.previousSocketData = socketData;
                  this.campaign = updatedCampaign;
                });
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
        dispatch(renderFlash('success', 'Query created'));
      })
      .catch(() => false);
  })

  onStopQuery = (evt) => {
    evt.preventDefault();

    const { removeSocket } = this;

    this.setState({ queryIsRunning: false });

    return removeSocket();
  }

  onTargetSelect = (selectedTargets) => {
    const { dispatch } = this.props;

    dispatch(setSelectedTargets(selectedTargets));

    return false;
  }

  onUpdateQuery = (formData) => {
    const { dispatch, query } = this.props;

    dispatch(queryActions.update(query, formData))
      .then(() => {
        dispatch(renderFlash('success', 'Query updated!'));
      });

    return false;
  };

  destroyCampaign = () => {
    const { campaign, dispatch } = this.props;
    const { destroy } = campaignActions;

    if (campaign) {
      this.campaign = null;
      dispatch(destroy(campaign));
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

  render () {
    const {
      onFetchTargets,
      onOsqueryTableSelect,
      onRunQuery,
      onSaveQueryFormSubmit,
      onStopQuery,
      onTargetSelect,
      onTextEditorInputChange,
      onUpdateQuery,
    } = this;
    const { queryIsRunning, targetsCount } = this.state;
    const {
      campaign,
      errors,
      query,
      selectedOsqueryTable,
      selectedTargets,
    } = this.props;

    return (
      <div className="has-sidebar">
        <QueryForm
          formData={query}
          handleSubmit={onSaveQueryFormSubmit}
          onFetchTargets={onFetchTargets}
          onOsqueryTableSelect={onOsqueryTableSelect}
          onRunQuery={onRunQuery}
          onStopQuery={onStopQuery}
          onTargetSelect={onTargetSelect}
          onUpdate={onUpdateQuery}
          queryIsRunning={queryIsRunning}
          selectedTargets={selectedTargets}
          serverErrors={errors}
          targetsCount={targetsCount}
          selectedOsqueryTable={selectedOsqueryTable}
        />
        {campaign && <QueryResultsTable campaign={campaign} />}
        <QuerySidePanel
          onOsqueryTableSelect={onOsqueryTableSelect}
          onTextEditorInputChange={onTextEditorInputChange}
          selectedOsqueryTable={selectedOsqueryTable}
        />
      </div>
    );
  }
}

const mapStateToProps = (state, { params }) => {
  const { id: queryID } = params;
  const { entities: campaigns } = entityGetter(state).get('campaigns');
  const reduxQuery = entityGetter(state).get('queries').findBy({ id: queryID });
  const { queryText, selectedOsqueryTable, selectedTargets } = state.components.QueryPages;
  const campaign = first(values(campaigns));
  const { errors } = state.entities.queries;
  const queryStub = { description: '', name: '', query: queryText };
  const query = reduxQuery || queryStub;

  return { campaign, errors, query, selectedOsqueryTable, selectedTargets };
};

export default connect(mapStateToProps)(QueryPage);
