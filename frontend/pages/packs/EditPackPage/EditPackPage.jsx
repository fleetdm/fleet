import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { filter, includes, isEqual, noop, size, find } from "lodash";
import { push } from "react-router-redux";

import permissionUtils from "utilities/permissions";
import deepDifference from "utilities/deep_difference";
import EditPackFormWrapper from "components/packs/EditPackFormWrapper";
import hostActions from "redux/nodes/entities/hosts/actions";
import hostInterface from "interfaces/host";
import labelActions from "redux/nodes/entities/labels/actions";
import teamActions from "redux/nodes/entities/teams/actions";
import labelInterface from "interfaces/label";
import teamInterface from "interfaces/team";
import packActions from "redux/nodes/entities/packs/actions";
import ScheduleQuerySidePanel from "components/side_panels/ScheduleQuerySidePanel";
import packInterface from "interfaces/pack";
import queryActions from "redux/nodes/entities/queries/actions";
import queryInterface from "interfaces/query";
import scheduledQueryInterface from "interfaces/scheduled_query";
import ScheduledQueriesListWrapper from "components/queries/ScheduledQueriesListWrapper";
import { renderFlash } from "redux/nodes/notifications/actions";
import scheduledQueryActions from "redux/nodes/entities/scheduled_queries/actions";
import stateEntityGetter from "redux/utilities/entityGetter";
import PATHS from "router/paths";

const baseClass = "edit-pack-page";
export class EditPackPage extends Component {
  static propTypes = {
    allQueries: PropTypes.arrayOf(queryInterface),
    dispatch: PropTypes.func,
    isEdit: PropTypes.bool,
    isLoadingPack: PropTypes.bool,
    isLoadingScheduledQueries: PropTypes.bool,
    pack: packInterface,
    packHosts: PropTypes.arrayOf(hostInterface),
    packID: PropTypes.string,
    packLabels: PropTypes.arrayOf(labelInterface),
    packTeams: PropTypes.arrayOf(teamInterface),
    scheduledQueries: PropTypes.arrayOf(scheduledQueryInterface),
    isBasicTier: PropTypes.bool,
  };

  static defaultProps = {
    dispatch: noop,
  };

  constructor(props) {
    super(props);

    this.state = {
      selectedQuery: null,
      selectedScheduledQuery: null,
      targetsCount: 0,
    };
  }

  componentWillMount() {
    const {
      allQueries,
      dispatch,
      isLoadingPack,
      pack,
      packHosts,
      packID,
      packLabels,
      packTeams,
      scheduledQueries,
    } = this.props;
    const { load } = packActions;
    const { loadAll } = queryActions;

    if (!pack && !isLoadingPack) {
      dispatch(load(packID));
    }

    if (pack) {
      if (!packHosts || packHosts.length !== pack.host_ids.length) {
        dispatch(hostActions.loadAll());
      }

      if (!packLabels || packLabels.length !== pack.label_ids.length) {
        dispatch(labelActions.loadAll());
      }

      if (!packTeams || packTeams.length !== pack.team_ids.length) {
        dispatch(teamActions.loadAll());
      }
    }

    if (!size(scheduledQueries)) {
      dispatch(scheduledQueryActions.loadAll({ id: packID }));
    }

    if (!size(allQueries)) {
      dispatch(loadAll());
    }

    return false;
  }

  componentWillReceiveProps({
    dispatch,
    pack,
    packHosts,
    packLabels,
    packTeams,
  }) {
    if (!isEqual(pack, this.props.pack)) {
      if (!packHosts || packHosts.length !== pack.host_ids.length) {
        dispatch(hostActions.loadAll());
      }

      if (!packLabels || packLabels.length !== pack.label_ids.length) {
        dispatch(labelActions.loadAll());
      }

      if (!packTeams || packTeams.length !== pack.team_ids.length) {
        dispatch(teamActions.loadAll());
      }
    }

    return false;
  }

  onCancelEditPack = () => {
    const { dispatch, isEdit, packID } = this.props;

    if (!isEdit) {
      return false;
    }

    return dispatch(push(PATHS.PACK({ id: packID })));
  };

  onFetchTargets = (query, targetsResponse) => {
    const { targets_count: targetsCount } = targetsResponse;

    this.setState({ targetsCount });

    return false;
  };

  onSelectQuery = (queryID) => {
    const { allQueries } = this.props;
    const selectedQuery = find(allQueries, { id: Number(queryID) });
    this.setState({ selectedQuery });

    return false;
  };

  onSelectScheduledQuery = (scheduledQuery) => {
    const { selectedScheduledQuery } = this.state;

    if (isEqual(scheduledQuery, selectedScheduledQuery)) {
      this.setState({ selectedScheduledQuery: null, selectedQuery: null });
    } else {
      this.onSelectQuery(scheduledQuery.query_id);
      this.setState({ selectedScheduledQuery: scheduledQuery });
    }

    return false;
  };

  onDblClickScheduledQuery = (scheduledQueryId) => {
    const { dispatch } = this.props;

    return dispatch(push(PATHS.EDIT_QUERY({ id: scheduledQueryId })));
  };

  onToggleEdit = () => {
    const { dispatch, isEdit, packID } = this.props;

    if (isEdit) {
      dispatch(push(PATHS.PACK({ id: packID })));
      dispatch(renderFlash("success", `Pack successfully updated.`));
      return null;
    }

    return dispatch(push(PATHS.EDIT_PACK({ id: packID })));
  };

  onUpdateScheduledQuery = (formData) => {
    const { dispatch } = this.props;
    const { selectedScheduledQuery } = this.state;
    const { update } = scheduledQueryActions;
    const updatedAttrs = deepDifference(formData, selectedScheduledQuery);

    dispatch(update(selectedScheduledQuery, updatedAttrs))
      .then(() => {
        this.setState({ selectedScheduledQuery: null, selectedQuery: null });
        dispatch(renderFlash("success", "Scheduled Query updated!"));
      })
      .catch(() => {
        dispatch(
          renderFlash("error", "Unable to update your Scheduled Query.")
        );
      });
  };

  handlePackFormSubmit = (formData) => {
    const { dispatch, pack } = this.props;
    const { update } = packActions;
    const updatedPack = deepDifference(formData, pack);
    return dispatch(update(pack, updatedPack))
      .then(() => {
        this.onToggleEdit();
      })
      .catch(() => {
        dispatch(
          renderFlash("error", `Could not update pack. Please try again.`)
        );
      });
  };

  handleRemoveScheduledQueries = (scheduledQueryIDs) => {
    const { destroy } = scheduledQueryActions;
    const { dispatch } = this.props;

    const promises = scheduledQueryIDs.map((id) => {
      return dispatch(destroy({ id }));
    });

    return Promise.all(promises).then(() => {
      this.setState({ selectedScheduledQuery: null, selectedQuery: null });
      dispatch(renderFlash("success", "Scheduled queries removed"));
    });
  };

  handleConfigurePackQuerySubmit = (formData) => {
    const { create } = scheduledQueryActions;
    const { dispatch, packID } = this.props;

    const scheduledQueryData = {
      ...formData,
      pack_id: packID,
    };

    dispatch(create(scheduledQueryData))
      // Will not render query name without declaring scheduledQueryData twice
      // eslint-disable-next-line @typescript-eslint/no-shadow
      .then((scheduledQueryData) => {
        dispatch(
          renderFlash(
            "success",
            `${scheduledQueryData.name} successfully scheduled to pack.`
          )
        );
      })
      .catch(() => {
        dispatch(renderFlash("error", "Unable to schedule your query."));
      });

    return false;
  };

  render() {
    const {
      handleConfigurePackQuerySubmit,
      handlePackFormSubmit,
      handleRemoveScheduledQueries,
      handleScheduledQueryFormSubmit,
      onCancelEditPack,
      onDblClickScheduledQuery,
      onFetchTargets,
      onSelectQuery,
      onSelectScheduledQuery,
      onToggleEdit,
      onUpdateScheduledQuery,
    } = this;
    const { targetsCount, selectedQuery, selectedScheduledQuery } = this.state;
    const {
      allQueries,
      isEdit,
      isLoadingPack,
      isLoadingScheduledQueries,
      pack,
      packHosts,
      packLabels,
      packTeams,
      scheduledQueries,
      isBasicTier,
    } = this.props;

    const packTargets = [...packHosts, ...packLabels, ...packTeams];

    if (!pack || isLoadingPack || isLoadingScheduledQueries) {
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
            packTargets={packTargets}
            targetsCount={targetsCount}
            isBasicTier={isBasicTier}
          />
          <ScheduledQueriesListWrapper
            onRemoveScheduledQueries={handleRemoveScheduledQueries}
            onScheduledQueryFormSubmit={handleScheduledQueryFormSubmit}
            onSelectScheduledQuery={onSelectScheduledQuery}
            onDblClickScheduledQuery={onDblClickScheduledQuery}
            scheduledQueries={scheduledQueries}
          />
        </div>
        <ScheduleQuerySidePanel
          onConfigurePackQuerySubmit={handleConfigurePackQuerySubmit}
          allQueries={allQueries}
          onFormCancel={onSelectScheduledQuery}
          onSelectQuery={onSelectQuery}
          onUpdateScheduledQuery={onUpdateScheduledQuery}
          selectedQuery={selectedQuery}
          selectedScheduledQuery={selectedScheduledQuery}
        />
      </div>
    );
  }
}

const mapStateToProps = (state, { params, route }) => {
  const entityGetter = stateEntityGetter(state);
  const isLoadingPack = state.entities.packs.loading;
  const { id: packID } = params;
  const pack = entityGetter.get("packs").findBy({ id: packID });
  const { entities: allQueries } = entityGetter.get("queries");
  const scheduledQueries = entityGetter
    .get("scheduled_queries")
    .where({ pack_id: packID });
  const isLoadingScheduledQueries = state.entities.scheduled_queries.loading;
  const isEdit = route.path === "edit";
  const packHosts = pack
    ? filter(state.entities.hosts.data, (host) => {
        return includes(pack.host_ids, host.id);
      })
    : [];
  const packLabels = pack
    ? filter(state.entities.labels.data, (label) => {
        return includes(pack.label_ids, label.id);
      })
    : [];
  const packTeams = pack
    ? filter(state.entities.teams.data, (team) => {
        return includes(pack.team_ids, team.id);
      })
    : [];
  const isBasicTier = permissionUtils.isBasicTier(state.app.config);

  return {
    allQueries,
    isEdit,
    isLoadingPack,
    isLoadingScheduledQueries,
    pack,
    packHosts,
    packID,
    packLabels,
    packTeams,
    scheduledQueries,
    isBasicTier,
  };
};

export default connect(mapStateToProps)(EditPackPage);
