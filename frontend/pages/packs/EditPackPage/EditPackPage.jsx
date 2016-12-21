import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { noop, size, find } from 'lodash';
import { push } from 'react-router-redux';

import EditPackFormWrapper from 'components/packs/EditPackFormWrapper';
import packActions from 'redux/nodes/entities/packs/actions';
import ScheduleQuerySidePanel from 'components/side_panels/ScheduleQuerySidePanel';
import packInterface from 'interfaces/pack';
import queryActions from 'redux/nodes/entities/queries/actions';
import queryInterface from 'interfaces/query';
import QueriesListWrapper from 'components/queries/QueriesListWrapper';
import { renderFlash } from 'redux/nodes/notifications/actions';
import scheduledQueryActions from 'redux/nodes/entities/scheduled_queries/actions';
import stateEntityGetter from 'redux/utilities/entityGetter';

const baseClass = 'edit-pack-page';

export class EditPackPage extends Component {
  static propTypes = {
    allQueries: PropTypes.arrayOf(queryInterface),
    dispatch: PropTypes.func,
    isEdit: PropTypes.bool,
    isLoadingPack: PropTypes.bool,
    isLoadingScheduledQueries: PropTypes.bool,
    pack: packInterface,
    packID: PropTypes.string,
    scheduledQueries: PropTypes.arrayOf(queryInterface),
  };

  static defaultProps = {
    dispatch: noop,
  };

  constructor (props) {
    super(props);

    this.state = {
      targetsCount: 0,
    };
  }

  componentDidMount () {
    const { allQueries, dispatch, isLoadingPack, pack, packID, scheduledQueries } = this.props;
    const { load } = packActions;
    const { loadAll } = queryActions;

    if (!pack && !isLoadingPack) {
      dispatch(load(packID));
    }

    if (!size(scheduledQueries)) {
      dispatch(scheduledQueryActions.loadAll({ id: packID }));
    }

    if (!size(allQueries)) {
      dispatch(loadAll());
    }

    return false;
  }

  onCancelEditPack = () => {
    const { dispatch, isEdit, packID } = this.props;

    if (!isEdit) {
      return false;
    }

    return dispatch(push(`/packs/${packID}`));
  }

  onFetchTargets = (query, targetsResponse) => {
    const { targets_count: targetsCount } = targetsResponse;

    this.setState({ targetsCount });

    return false;
  }

  onSelectQuery = (query) => {
    const { allQueries } = this.props;
    const selectedQuery = find(allQueries, { id: Number(query) });
    this.setState({ selectedQuery });

    return false;
  }

  onToggleEdit = () => {
    const { dispatch, isEdit, packID } = this.props;

    if (isEdit) {
      return dispatch(push(`/packs/${packID}`));
    }

    return dispatch(push(`/packs/${packID}/edit`));
  }

  handlePackFormSubmit = (formData) => {
    const { dispatch } = this.props;
    const { update } = packActions;

    return dispatch(update(formData));
  }

  handleRemoveScheduledQueries = (scheduledQueryIDs) => {
    const { destroy } = scheduledQueryActions;
    const { dispatch } = this.props;

    const promises = scheduledQueryIDs.map((id) => {
      return dispatch(destroy({ id }));
    });

    return Promise.all(promises)
      .then(() => {
        dispatch(renderFlash('success', 'Scheduled queries removed'));
      });
  }

  handleConfigurePackQuerySubmit = (formData) => {
    const { create } = scheduledQueryActions;
    const { dispatch, packID } = this.props;
    const scheduledQueryData = {
      ...formData,
      snapshot: formData.logging_type === 'snapshot',
      pack_id: packID,
    };

    dispatch(create(scheduledQueryData))
      .then(() => {
        dispatch(renderFlash('success', 'Query scheduled!'));
      })
      .catch(() => {
        dispatch(renderFlash('error', 'Unable to schedule your query.'));
      });

    return false;
  }

  render () {
    const {
      handleConfigurePackQuerySubmit,
      handlePackFormSubmit,
      handleRemoveScheduledQueries,
      handleScheduledQueryFormSubmit,
      onCancelEditPack,
      onFetchTargets,
      onSelectQuery,
      onToggleEdit,
    } = this;
    const { targetsCount, selectedQuery } = this.state;
    const { allQueries, isEdit, isLoadingScheduledQueries, pack, scheduledQueries } = this.props;

    if (!pack || isLoadingScheduledQueries) {
      return false;
    }

    return (
      <div className={`${baseClass} has-sidebar`}>
        <div className={`${baseClass}__content`}>
          <EditPackFormWrapper
            className={`${baseClass}__pack-form body-wrap`}
            handleSubmit={handlePackFormSubmit}
            isEdit={isEdit}
            onCancelEditPack={onCancelEditPack}
            onEditPack={onToggleEdit}
            onFetchTargets={onFetchTargets}
            pack={pack}
            targetsCount={targetsCount}
          />
          <QueriesListWrapper
            onRemoveScheduledQueries={handleRemoveScheduledQueries}
            onScheduledQueryFormSubmit={handleScheduledQueryFormSubmit}
            scheduledQueries={scheduledQueries}
          />
        </div>
        <ScheduleQuerySidePanel
          onConfigurePackQuerySubmit={handleConfigurePackQuerySubmit}
          allQueries={allQueries}
          onSelectQuery={onSelectQuery}
          selectedQuery={selectedQuery}
        />
      </div>
    );
  }
}

const mapStateToProps = (state, { params, route }) => {
  const entityGetter = stateEntityGetter(state);
  const isLoadingPack = state.entities.packs.loading;
  const { id: packID } = params;
  const pack = entityGetter.get('packs').findBy({ id: packID });
  const { entities: allQueries } = entityGetter.get('queries');
  const scheduledQueries = entityGetter.get('scheduled_queries').where({ pack_id: packID });
  const isLoadingScheduledQueries = state.entities.scheduled_queries.loading;
  const isEdit = route.path === 'edit';

  return {
    allQueries,
    isEdit,
    isLoadingPack,
    isLoadingScheduledQueries,
    pack,
    packID,
    scheduledQueries,
  };
};

export default connect(mapStateToProps)(EditPackPage);
