import React, { useState, useEffect, useContext } from "react";
import { useQuery, useMutation } from "react-query";
import { Params } from "react-router/lib/Router";

import PropTypes from "prop-types";
import { connect } from "react-redux";
import { filter, includes, isEqual, noop, size, find } from "lodash";
import { push } from "react-router-redux";

// second grouping
// @ts-ignore
import { IConfig } from "interfaces/config";
import { IError } from "interfaces/errors";
import { IHost } from "interfaces/host";
import { ILabel } from "interfaces/label";
import { IPack } from "interfaces/pack";
import { IQuery } from "interfaces/query";
import { IScheduledQuery } from "interfaces/scheduled_query";
import { ITeam } from "interfaces/team";
import { AppContext } from "context/app";

import configAPI from "services/entities/config";
import hostAPI from "services/entities/hosts";
import labelAPI from "services/entities/labels";
import packAPI from "services/entities/packs";
import queryAPI from "services/entities/queries";
import scheduledqueryAPI from "services/entities/scheduled_queries";
import teamAPI from "services/entities/teams";

// @ts-ignore
import deepDifference from "utilities/deep_difference";
// @ts-ignore
import EditPackFormWrapper from "components/packs/EditPackFormWrapper";
// @ts-ignore
import ScheduleQuerySidePanel from "components/side_panels/ScheduleQuerySidePanel";
// @ts-ignore
import ScheduledQueriesListWrapper from "components/queries/ScheduledQueriesListWrapper";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

// import permissionUtils from "utilities/permissions";
// import hostActions from "redux/nodes/entities/hosts/actions";
// import hostInterface from "interfaces/host";
// import labelActions from "redux/nodes/entities/labels/actions";
// import teamActions from "redux/nodes/entities/teams/actions";
// import labelInterface from "interfaces/label";
// import teamInterface from "interfaces/team";
// import packActions from "redux/nodes/entities/packs/actions";
// import queryActions from "redux/nodes/entities/queries/actions";
// import scheduledQueryInterface from "interfaces/scheduled_query";
// import scheduledQueryActions from "redux/nodes/entities/scheduled_queries/actions";
// import stateEntityGetter from "redux/utilities/entityGetter";
import PATHS from "router/paths";

// NEW TSX CODE
interface IEditPacksPageProps {
  router: any;
  params: Params;
  location: any; // TODO: find Location type
}
interface IRootState {
  app: {
    config: IConfig;
  };
  entities: {
    packs: {
      loading: boolean; // done
      data: IPack[];
      errors: IError[];
    };
    hosts: {
      isLoading: boolean;
      data: IHost[];
      errors: IError[];
    };
    queries: {
      isLoading: boolean;
      data: IQuery[];
      errors: IError[];
    };
    teams: {
      isLoading: boolean;
      data: ITeam[];
      errors: IError[];
    };
    labels: {
      isLoading: boolean;
      data: ILabel[];
      errors: IError[];
    };
  };
}

interface IStoredPackResponse {
  pack: IPack;
}

interface IStoredFleetQueriesResponse {
  fleetQueries: IQuery[];
}

interface IStoredScheduledQueriesResponse {
  scheduledQueries: IScheduledQuery[];
}

interface IStoredLabelsResponse {
  labels: ILabel[];
}
interface IStoredHostsResponse {
  hosts: IHost[];
}

interface IStoredTeamsResponse {
  teams: ITeam[];
}

const baseClass = "edit-pack-page";

const EditPacksPage = ({
  router, // only needed if I need to navigate to another page from this page
  params: { id: paramsPackId },
  location: { query: URLQueryString }, // might need this if there's team filters
}: IEditPacksPageProps): JSX.Element => {
  const packId = parseInt(paramsPackId, 10);

  const [pack, setPack] = useState<IPack | null>(null);

  // react-query uses your own api and gives you different states of loading data
  // can set to retreive data based on different properties
  const {
    isLoading: isStoredPackLoading,
    data: storedPack, // only returns pack and not response wrapping
    error: storedPackError,
  } = useQuery<IStoredPackResponse, Error, IPack>(
    ["pack", packId],
    () => packAPI.load(packId),
    {
      enabled: !!packId, // doesn't run unless ID is given, unneeded but extra precaution
      select: (data: IStoredPackResponse) => data.pack,
    }
  );

  const {
    isLoading: isFleetQueriesLoading,
    data: fleetQueries,
    error: fleetQueriesError,
  } = useQuery<IStoredFleetQueriesResponse, Error, IQuery[]>(
    ["fleet queries"], // use single string or array of strings can be named anything
    () => queryAPI.loadAll(),
    {
      select: (data: IStoredFleetQueriesResponse) => data.fleetQueries,
    }
  );

  if (storedPack) {
    const {
      isLoading: isScheduledQueriesLoading,
      data: scheduledQueries,
      error: scheduledQueriesError,
    } = useQuery<IStoredScheduledQueriesResponse, Error, IScheduledQuery[]>(
      ["scheduled queries"], // use single string or array of strings can be named anything
      () => scheduledqueryAPI.loadAll(storedPack),
      {
        select: (data: IStoredScheduledQueriesResponse) =>
          data.scheduledQueries,
      }
    );
  }

  const {
    isLoading: isLabelsLoading,
    data: labels,
    error: packLabelsError,
  } = useQuery<IStoredLabelsResponse, Error, ILabel[]>(
    ["pack labels"], // use single string or array of strings can be named anything
    () => labelAPI.loadAll(),
    {
      select: (data: IStoredLabelsResponse) => data.labels,
    }
  );

  const packLabels = storedPack
    ? filter(labels, (label) => {
        return includes(storedPack.label_ids, label.id);
      })
    : [];

  const {
    isLoading: isHostsLoading,
    data: hosts,
    error: hostsError,
  } = useQuery<IStoredHostsResponse, Error, IHost[]>(
    ["pack labels"],
    () => hostAPI.loadAll(undefined), // is this how this works?
    {
      select: (data: IStoredHostsResponse) => data.hosts,
    }
  );

  const packHosts = storedPack
    ? filter(hosts, (host) => {
        return includes(storedPack.host_ids, host.id);
      })
    : [];

  const {
    isLoading: isTeamsLoading,
    data: teams,
    error: teamsError,
  } = useQuery<IStoredTeamsResponse, Error, ITeam[]>(
    ["pack labels"],
    () => teamAPI.loadAll(),
    {
      select: (data: IStoredTeamsResponse) => data.teams,
    }
  );

  const packTeams = pack
    ? filter(teams, (team) => {
        return includes(pack.team_ids, team.id);
      })
    : [];
  const { isPremiumTier } = useContext(AppContext);

  return <></>;
};

// EVERYTHING BELOW HERE IS UNTOUCHED CLASS JSX

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
    isPremiumTier: PropTypes.bool,
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
    } = this.props;
    const { load } = packActions;
    const { loadAll } = queryActions;

    if (!isLoadingPack) {
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

    dispatch(scheduledQueryActions.loadAll({ id: packID }));

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
    const { destroy, loadAll } = scheduledQueryActions;
    const { dispatch } = this.props;

    const promises = scheduledQueryIDs.map((id) => {
      return dispatch(destroy({ id }));
    });

    const queryOrQueries = scheduledQueryIDs.length === 1 ? "query" : "queries";

    return Promise.all(promises)
      .then(() => {
        this.setState({ selectedScheduledQuery: null, selectedQuery: null });
        dispatch(
          renderFlash(
            "success",
            `Scheduled ${queryOrQueries} removed from pack.`
          )
        );
      })
      .catch(() => {
        dispatch(
          renderFlash(
            "error",
            `Could not remove ${queryOrQueries}. Please try again.`
          )
        );
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
      isPremiumTier,
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
            isPremiumTier={isPremiumTier}
          />
          <ScheduledQueriesListWrapper
            onRemoveScheduledQueries={handleRemoveScheduledQueries}
            onScheduledQueryFormSubmit={handleScheduledQueryFormSubmit}
            onSelectScheduledQuery={onSelectScheduledQuery}
            onDblClickScheduledQuery={onDblClickScheduledQuery}
            scheduledQueries={scheduledQueries}
            packId={pack.id}
            isLoadingScheduledQueries={isLoadingScheduledQueries}
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
  // const entityGetter = stateEntityGetter(state); // done
  // const isLoadingPack = state.entities.packs.loading; // done
  // const { id: packID } = params; // done
  // const pack = entityGetter.get("packs").findBy({ id: packID }); //done
  // const { entities: allQueries } = entityGetter.get("queries"); // done
  // const scheduledQueries = entityGetter
  //   .get("scheduled_queries")
  //   .where({ pack_id: packID }); // done
  // const isLoadingScheduledQueries = state.entities.scheduled_queries.loading; // done
  // const isEdit = route.path === "edit"; // no more edit button
  // const packHosts = pack
  //   ? filter(state.entities.hosts.data, (host) => {
  //       return includes(pack.host_ids, host.id);
  //     })
  //   : []; // done
  // const packLabels = pack
  //   ? filter(state.entities.labels.data, (label) => {
  //       return includes(pack.label_ids, label.id);
  //     })
  //   : []; // done
  const packTeams = pack
    ? filter(state.entities.teams.data, (team) => {
        return includes(pack.team_ids, team.id);
      })
    : []; // done
  // const isPremiumTier = permissionUtils.isPremiumTier(state.app.config); // done

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
    isPremiumTier,
  };
};

export default connect(mapStateToProps)(EditPackPage);
