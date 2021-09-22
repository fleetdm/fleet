import React, { useState, useEffect, useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { Params } from "react-router/lib/Router";

import { filter, includes } from "lodash";
import { useDispatch } from "react-redux";
import { push } from "react-router-redux";

// second grouping
// @ts-ignore
import { IConfig } from "interfaces/config";
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
import { renderFlash } from "redux/nodes/notifications/actions";
// @ts-ignore
import debounce from "utilities/debounce";
import PATHS from "router/paths";
// @ts-ignore
import deepDifference from "utilities/deep_difference";

import EditPackForm from "components/forms/packs/EditPackForm";
import PackQueriesListWrapper from "components/queries/PackQueriesListWrapper";
import PackQueryEditorModal from "./components/PackQueryEditorModal";
import RemovePackQueryModal from "./components/RemovePackQueryModal";
import { ITargetsAPIResponse } from "interfaces/target";

interface IEditPacksPageProps {
  router: any;
  params: Params;
  location: any; // TODO: find Location type
}
interface IRootState {
  app: {
    config: IConfig;
  };
}

interface IPackQueryFormData {
  interval: number;
  name?: string;
  shard: number;
  query?: string;
  query_id?: number;
  removed: boolean;
  snapshot: boolean;
  pack_id: number;
  platform: string;
  version: string;
}

interface IStoredFleetQueriesResponse {
  queries: IQuery[];
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
  // DATA AND API CALLS
  const { isPremiumTier } = useContext(AppContext);

  const dispatch = useDispatch();
  const packId: number = parseInt(paramsPackId, 10);

  const [targetsCount, setTargetsCount] = useState<number>(0);
  const [
    showPackQueryEditorModal,
    setShowPackQueryEditorModal,
  ] = useState<boolean>(false);
  const [showEditPackQueryModal, setShowEditPackQueryModal] = useState<boolean>(
    false
  );
  const [
    showRemovePackQueryModal,
    setShowRemovePackQueryModal,
  ] = useState<boolean>(false);
  const [selectedPackQuery, setSelectedPackQuery] = useState<IScheduledQuery>();
  const [selectedPackQueryIds, setSelectedPackQueryIds] = useState<
    number[] | never[]
  >([]);

  const [storedPack, setStoredPack] = useState<IPack | undefined>();
  const [isStoredPackLoading, setIsStoredPackLoading] = useState(true);
  const [isStoredPackLoadingError, setIsStoredPackLoadingError] = useState(
    false
  );

  const [storedPackQueries, setStoredPackQueries] = useState<
    IScheduledQuery[] | never[]
  >([]);
  const [isStoredPackQueriesLoading, setIsStoredPackQueriesLoading] = useState(
    true
  );
  const [
    isStoredPackQueriesLoadingError,
    setIsStoredPackQueriesLoadingError,
  ] = useState(false);

  const getPack = useCallback(async () => {
    setIsStoredPackLoading(true);
    try {
      const response = await packAPI.load(packId);
      setStoredPack(response.pack);
    } catch (error) {
      console.log(error);
      setIsStoredPackLoadingError(true);
    } finally {
      setIsStoredPackLoading(false);
    }
  }, [dispatch]);

  const getPackQueries = useCallback(async () => {
    setIsStoredPackQueriesLoading(true);
    try {
      const response = await scheduledqueryAPI.loadAll(packId);
      setStoredPackQueries(response.scheduled);
    } catch (error) {
      console.log(error);
      setIsStoredPackQueriesLoadingError(true);
    } finally {
      setIsStoredPackQueriesLoading(false);
    }
  }, [dispatch]);

  useEffect(() => {
    getPack();
    getPackQueries();
  }, [getPackQueries, getPack]);

  const {
    isLoading: isFleetQueriesLoading,
    data: fleetQueries,
    error: fleetQueriesError,
  } = useQuery<IStoredFleetQueriesResponse, Error, IQuery[]>(
    ["fleet queries"], // use single string or array of strings can be named anything
    () => queryAPI.loadAll(),
    {
      select: (data: IStoredFleetQueriesResponse) => data.queries,
    }
  );

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
    ["all hosts"], // use single string or array of strings can be named anything
    () => hostAPI.loadAll(undefined),
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

  const packTeams = storedPack
    ? filter(teams, (team) => {
        return includes(storedPack.team_ids, team.id);
      })
    : [];

  const packTargets = [...packHosts, ...packLabels, ...packTeams];

  const onCancelEditPack = () => {
    return dispatch(push(PATHS.MANAGE_PACKS));
  };

  const onFetchTargets = (
    query: IQuery,
    targetsResponse: ITargetsAPIResponse
  ) => {
    const { targets_count } = targetsResponse;

    setTargetsCount(targets_count);

    return false;
  };

  const togglePackQueryEditorModal = useCallback(() => {
    setSelectedPackQuery(undefined); // create modal renders
    setShowPackQueryEditorModal(!showPackQueryEditorModal);
    console.log("togglePackQueryEditorModal clicked!");
  }, [showPackQueryEditorModal, setShowPackQueryEditorModal]);

  const toggleEditPackQueryModal = useCallback(() => {
    setShowEditPackQueryModal(!showEditPackQueryModal);
  }, [showEditPackQueryModal, setShowEditPackQueryModal]);

  const toggleRemovePackQueryModal = useCallback(() => {
    setShowRemovePackQueryModal(!showRemovePackQueryModal);
  }, [showRemovePackQueryModal, setShowRemovePackQueryModal]);

  const onEditPackQueryClick = (selectedQuery: any): void => {
    togglePackQueryEditorModal();
    setSelectedPackQuery(selectedQuery); // edit modal renders
  };

  const onRemovePackQueriesClick = (selectedTableQueryIds: any): void => {
    toggleRemovePackQueryModal();
    setSelectedPackQueryIds(selectedTableQueryIds);
  };

  const handlePackFormSubmit = (formData: any) => {
    const updatedPack = deepDifference(formData, storedPack);
    packAPI
      .update(packId, updatedPack)
      .then(() => {
        dispatch(renderFlash("success", `Successfully updated this pack.`));
      })
      .catch(() => {
        dispatch(
          renderFlash("error", `Could not update pack. Please try again.`)
        );
      });
  };

  const onPackQueryEditorSubmit = useCallback(
    (formData: IPackQueryFormData, editQuery: IScheduledQuery | undefined) => {
      const request = editQuery
        ? scheduledqueryAPI.update(editQuery, formData)
        : scheduledqueryAPI.create(formData);
      request
        .then(() => {
          dispatch(renderFlash("success", `Successfully updated this pack.`));
        })
        .catch(() => {
          dispatch(
            renderFlash(
              "error",
              "Could not update this pack. Please try again."
            )
          );
        })
        .finally(() => {
          togglePackQueryEditorModal();
          getPackQueries();
        });
      return false;
    },
    [dispatch, getPackQueries, togglePackQueryEditorModal]
  );

  const onRemovePackQuerySubmit = useCallback(() => {
    const queryOrQueries =
      selectedPackQueryIds.length === 1 ? "query" : "queries";

    const promises = selectedPackQueryIds.map((id: number) => {
      return scheduledqueryAPI.destroy(id);
    });

    return Promise.all(promises)
      .then(() => {
        dispatch(
          renderFlash(
            "success",
            `Successfully removed ${queryOrQueries} from this pack.`
          )
        );
      })
      .catch(() => {
        dispatch(
          renderFlash(
            "error",
            `Unable to remove ${queryOrQueries} from this pack. Please try again.`
          )
        );
      })
      .finally(() => {
        toggleRemovePackQueryModal();
        getPackQueries();
      });
  }, [
    dispatch,
    getPackQueries,
    selectedPackQueryIds,
    toggleRemovePackQueryModal,
  ]);

  return (
    <div className={`${baseClass}__content`}>
      {storedPack && (
        <EditPackForm
          className={`${baseClass}__pack-form body-wrap`}
          handleSubmit={handlePackFormSubmit}
          onCancelEditPack={onCancelEditPack}
          onFetchTargets={onFetchTargets}
          formData={{ ...storedPack, targets: packTargets }}
          targetsCount={targetsCount}
          isPremiumTier={isPremiumTier}
        />
      )}
      <PackQueriesListWrapper
        onAddPackQuery={togglePackQueryEditorModal}
        onEditPackQuery={onEditPackQueryClick}
        onRemovePackQueries={onRemovePackQueriesClick}
        onPackQueryFormSubmit={onPackQueryEditorSubmit}
        scheduledQueries={storedPackQueries}
        packId={packId}
        isLoadingPackQueries={isStoredPackQueriesLoading}
      />
      {showPackQueryEditorModal && fleetQueries && (
        <PackQueryEditorModal
          onCancel={togglePackQueryEditorModal}
          onPackQueryFormSubmit={onPackQueryEditorSubmit}
          allQueries={fleetQueries}
          editQuery={selectedPackQuery}
          packId={packId}
        />
      )}
      {showRemovePackQueryModal && fleetQueries && (
        <RemovePackQueryModal
          onCancel={toggleRemovePackQueryModal}
          onSubmit={onRemovePackQuerySubmit}
          selectedQueries={selectedPackQuery}
        />
      )}
    </div>
  );
};

export default EditPacksPage;
