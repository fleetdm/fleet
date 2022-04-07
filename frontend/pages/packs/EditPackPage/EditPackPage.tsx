import React, { useState, useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";

import { IPack } from "interfaces/pack";
import { IQuery } from "interfaces/query";
import {
  IPackQueryFormData,
  IScheduledQuery,
} from "interfaces/scheduled_query";
import { ITarget, ITargetsAPIResponse } from "interfaces/target";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import packsAPI from "services/entities/packs";
import queriesAPI from "services/entities/queries";
import scheduledqueriesAPI from "services/entities/scheduled_queries";

import PATHS from "router/paths"; // @ts-ignore
import deepDifference from "utilities/deep_difference";

import EditPackForm from "components/forms/packs/EditPackForm";
import PackQueryEditorModal from "./components/PackQueryEditorModal";
import RemovePackQueryModal from "./components/RemovePackQueryModal";

interface IEditPacksPageProps {
  router: InjectedRouter; // v3
  params: Params;
}

interface IStoredFleetQueriesResponse {
  queries: IQuery[];
}

interface IStoredPackResponse {
  pack: IPack;
}

interface IStoredPackQueriesResponse {
  scheduled: IScheduledQuery[];
}

interface IFormData {
  name?: string;
  description?: string;
  targets?: ITarget[];
}

const baseClass = "edit-pack-page";

const EditPacksPage = ({
  router,
  params: { id: paramsPackId },
}: IEditPacksPageProps): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const packId: number = parseInt(paramsPackId, 10);

  const { data: fleetQueries } = useQuery<
    IStoredFleetQueriesResponse,
    Error,
    IQuery[]
  >(["fleet queries"], () => queriesAPI.loadAll(), {
    select: (data: IStoredFleetQueriesResponse) => data.queries,
  });

  const { data: storedPack } = useQuery<IStoredPackResponse, Error, IPack>(
    ["stored pack"],
    () => packsAPI.load(packId),
    {
      select: (data: IStoredPackResponse) => data.pack,
    }
  );

  const {
    data: storedPackQueries,
    isLoading: isStoredPackQueriesLoading,
    refetch: refetchStoredPackQueries,
  } = useQuery<IStoredPackQueriesResponse, Error, IScheduledQuery[]>(
    ["stored pack queries"],
    () => scheduledqueriesAPI.loadAll(packId),
    {
      select: (data: IStoredPackQueriesResponse) => data.scheduled,
    }
  );

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

  const packTargets = storedPack
    ? [
        ...storedPack.hosts.map((host) => ({
          ...host,
          target_type: "hosts",
        })),
        ...storedPack.labels.map((label) => ({
          ...label,
          target_type: "labels",
        })),
        ...storedPack.teams.map((team) => ({
          ...team,
          target_type: "teams",
        })),
      ]
    : [];

  const onCancelEditPack = () => {
    return router.push(PATHS.MANAGE_PACKS);
  };

  const onFetchTargets = useCallback(
    (query: IQuery, targetsResponse: ITargetsAPIResponse) => {
      const { targets_count } = targetsResponse;
      setTargetsCount(targets_count);

      return false;
    },
    []
  );

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

  const handlePackFormSubmit = (formData: IFormData) => {
    const updatedPack = deepDifference(formData, storedPack);
    packsAPI
      .update(packId, updatedPack)
      .then(() => {
        router.push(PATHS.MANAGE_PACKS);
        renderFlash("success", `Successfully updated this pack.`);
      })
      .catch((response) => {
        if (
          response.errors[0].reason.slice(0, 27) ===
          "Error 1062: Duplicate entry"
        ) {
          renderFlash(
            "error",
            "Unable to update pack. Pack names must be unique."
          );
        } else {
          renderFlash("error", `Could not update pack. Please try again.`);
        }
      });
  };

  const onPackQueryEditorSubmit = (
    formData: IPackQueryFormData,
    editQuery: IScheduledQuery | undefined
  ) => {
    const request = editQuery
      ? scheduledqueriesAPI.update(editQuery, formData)
      : scheduledqueriesAPI.create(formData);
    request
      .then(() => {
        renderFlash("success", `Successfully updated this pack.`);
      })
      .catch(() => {
        renderFlash("error", "Could not update this pack. Please try again.");
      })
      .finally(() => {
        togglePackQueryEditorModal();
        refetchStoredPackQueries();
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
        renderFlash(
          "success",
          `Successfully removed ${queryOrQueries} from this pack.`
        );
      })
      .catch(() => {
        renderFlash(
          "error",
          `Unable to remove ${queryOrQueries} from this pack. Please try again.`
        );
      })
      .finally(() => {
        toggleRemovePackQueryModal();
        refetchStoredPackQueries();
      });
  };

  return (
    <div className={`${baseClass}__content`}>
      {storedPack && storedPackQueries && (
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
          scheduledQueries={storedPackQueries}
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
          selectedQuery={selectedPackQuery}
          selectedQueryIds={selectedPackQueryIds}
        />
      )}
    </div>
  );
};

export default EditPacksPage;
