import React, { useState, useEffect, useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { Params } from "react-router/lib/Router";

import { filter, includes } from "lodash";
import { useDispatch } from "react-redux";
import { push } from "react-router-redux";

// @ts-ignore
import { IConfig } from "interfaces/config";
import { IHost } from "interfaces/host";
import { ILabel } from "interfaces/label";
import { IPack } from "interfaces/pack";
import { IQuery } from "interfaces/query";
import {
  IPackQueryFormData,
  IScheduledQuery,
} from "interfaces/scheduled_query";
import { ITargetsAPIResponse } from "interfaces/target";
import { ITeam } from "interfaces/team";
import { AppContext } from "context/app";

import hostsAPI from "services/entities/hosts";
import labelsAPI from "services/entities/labels";
import packsAPI from "services/entities/packs";
import queriesAPI from "services/entities/queries";
import scheduledqueriesAPI from "services/entities/scheduled_queries";
import teamsAPI from "services/entities/teams";

// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import PATHS from "router/paths";
// @ts-ignore
import deepDifference from "utilities/deep_difference";

import EditPackForm from "components/forms/packs/EditPackForm";
import PackQueryEditorModal from "./components/PackQueryEditorModal";
import RemovePackQueryModal from "./components/RemovePackQueryModal";

interface IEditPacksPageProps {
  router: any;
  params: Params;
  location: any; // TODO: find Location type
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
  router,
  params: { id: paramsPackId },
  location: { query: URLQueryString },
}: IEditPacksPageProps): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);

  const dispatch = useDispatch();
  const packId: number = parseInt(paramsPackId, 10);

  const [targetsCount, setTargetsCount] = useState<number>(0);
  const [
    showPackQueryEditorModal,
    setShowPackQueryEditorModal,
  ] = useState<boolean>(false);
  const [
    showRemovePackQueryModal,
    setShowRemovePackQueryModal,
  ] = useState<boolean>(false);
  const [selectedPackQuery, setSelectedPackQuery] = useState<IScheduledQuery>();
  const [selectedPackQueryIds, setSelectedPackQueryIds] = useState<
    number[] | never[]
  >([]);

  const [storedPack, setStoredPack] = useState<IPack | undefined>();
  const [isStoredPackLoading, setIsStoredPackLoading] = useState<boolean>(true);
  const [
    isStoredPackLoadingError,
    setIsStoredPackLoadingError,
  ] = useState<boolean>(false);

  const [storedPackQueries, setStoredPackQueries] = useState<
    IScheduledQuery[] | never[]
  >([]);
  const [
    isStoredPackQueriesLoading,
    setIsStoredPackQueriesLoading,
  ] = useState<boolean>(true);
  const [
    isStoredPackQueriesLoadingError,
    setIsStoredPackQueriesLoadingError,
  ] = useState<boolean>(false);

  const getPack = useCallback(async () => {
    setIsStoredPackLoading(true);
    try {
      const response = await packsAPI.load(packId);
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
      const response = await scheduledqueriesAPI.loadAll(packId);
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
    ["fleet queries"],
    () => queriesAPI.loadAll(),
    {
      select: (data: IStoredFleetQueriesResponse) => data.queries,
    }
  );

  const {
    isLoading: isLabelsLoading,
    data: labels,
    error: packLabelsError,
  } = useQuery<IStoredLabelsResponse, Error, ILabel[]>(
    ["pack labels"],
    () => labelsAPI.loadAll(),
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
    ["all hosts"],
    () => hostsAPI.loadAll(undefined),
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
    () => teamsAPI.loadAll(),
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

  const togglePackQueryEditorModal = () => {
    setSelectedPackQuery(undefined); // create modal renders
    setShowPackQueryEditorModal(!showPackQueryEditorModal);
  };

  const toggleRemovePackQueryModal = () => {
    setShowRemovePackQueryModal(!showRemovePackQueryModal);
  };

  const onEditPackQueryClick = (selectedQuery: IScheduledQuery): void => {
    togglePackQueryEditorModal();
    setSelectedPackQuery(selectedQuery); // edit modal renders
  };

  const onRemovePackQueriesClick = (selectedTableQueryIds: number[]): void => {
    toggleRemovePackQueryModal();
    setSelectedPackQueryIds(selectedTableQueryIds);
  };

  const handlePackFormSubmit = useCallback(
    (formData: any) => {
      const updatedPack = deepDifference(formData, storedPack);
      console.log("handlePackFormSubmit formData", formData);
      console.log("handlePackFormSubmit storedPack", storedPack);
      console.log("handlePackFormSubmit updatedPack", updatedPack);
      debugger;
      packsAPI
        .update(packId, updatedPack)
        .then(() => {
          dispatch(renderFlash("success", `Successfully updated this pack.`));
        })
        .catch(() => {
          dispatch(
            renderFlash("error", `Could not update pack. Please try again.`)
          );
        });
    },
    [storedPack, packId]
  );

  const onPackQueryEditorSubmit = (
    formData: IPackQueryFormData,
    editQuery: IScheduledQuery | undefined
  ) => {
    const request = editQuery
      ? scheduledqueriesAPI.update(editQuery, formData)
      : scheduledqueriesAPI.create(formData);
    request
      .then(() => {
        dispatch(renderFlash("success", `Successfully updated this pack.`));
      })
      .catch(() => {
        dispatch(
          renderFlash("error", "Could not update this pack. Please try again.")
        );
      })
      .finally(() => {
        togglePackQueryEditorModal();
        getPackQueries();
      });
    return false;
  };

  const onRemovePackQuerySubmit = () => {
    const queryOrQueries =
      selectedPackQueryIds.length === 1 ? "query" : "queries";

    const promises = selectedPackQueryIds.map((id: number) => {
      return scheduledqueriesAPI.destroy(id);
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
  };

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
          onAddPackQuery={togglePackQueryEditorModal}
          onEditPackQuery={onEditPackQueryClick}
          onRemovePackQueries={onRemovePackQueriesClick}
          onPackQueryFormSubmit={onPackQueryEditorSubmit}
          scheduledQueries={storedPackQueries}
          packId={packId}
          isLoadingPackQueries={isStoredPackQueriesLoading}
        />
      )}
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
