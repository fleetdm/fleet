import React, { useState, useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import { IPack, IStoredPackResponse } from "interfaces/pack";
import { IQuery } from "interfaces/query";
import {
  IPackQueryFormData,
  IScheduledQuery,
  IStoredScheduledQueriesResponse,
} from "interfaces/scheduled_query";
import { ITarget, ITargetsAPIResponse } from "interfaces/target";
import {
  IQueryKeyQueriesLoadAll,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import { getErrorReason } from "interfaces/errors";

import packsAPI from "services/entities/packs";
import queriesAPI, { IQueriesResponse } from "services/entities/queries";
import scheduledQueriesAPI from "services/entities/scheduled_queries";

import PATHS from "router/paths";
// @ts-ignore
import deepDifference from "utilities/deep_difference";

import BackLink from "components/BackLink";
import EditPackForm from "components/forms/packs/EditPackForm";
import MainContent from "components/MainContent";
import PackQueryEditorModal from "./components/PackQueryEditorModal";
import RemovePackQueryModal from "./components/RemovePackQueryModal";

interface IEditPacksPageProps {
  router: InjectedRouter; // v3
  params: Params;
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

  const { data: queries } = useQuery<
    IQueriesResponse,
    Error,
    ISchedulableQuery[],
    IQueryKeyQueriesLoadAll[]
  >(
    [{ scope: "queries", teamId: undefined }],
    ({ queryKey }) => queriesAPI.loadAll(queryKey[0]),
    {
      select: (data) => data.queries,
    }
  );

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
  } = useQuery<IStoredScheduledQueriesResponse, Error, IScheduledQuery[]>(
    ["stored pack queries"],
    () => scheduledQueriesAPI.loadAll(packId),
    {
      select: (data: IStoredScheduledQueriesResponse) => data.scheduled,
    }
  );

  const [targetsCount, setTargetsCount] = useState(0);
  const [showPackQueryEditorModal, setShowPackQueryEditorModal] = useState(
    false
  );
  const [showRemovePackQueryModal, setShowRemovePackQueryModal] = useState(
    false
  );
  const [selectedPackQuery, setSelectedPackQuery] = useState<IScheduledQuery>();
  const [selectedPackQueryIds, setSelectedPackQueryIds] = useState<
    number[] | never[]
  >([]);
  const [isUpdatingPack, setIsUpdatingPack] = useState(false);

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
    setIsUpdatingPack(true);
    const updatedPack = deepDifference(formData, storedPack);
    packsAPI
      .update(packId, updatedPack)
      .then(() => {
        router.push(PATHS.MANAGE_PACKS);
        renderFlash("success", `Successfully updated this pack.`);
      })
      .catch((e) => {
        if (
          getErrorReason(e, {
            reasonIncludes: "Duplicate entry",
          })
        ) {
          renderFlash(
            "error",
            "Unable to update pack. Pack names must be unique."
          );
        } else {
          renderFlash("error", `Could not update pack. Please try again.`);
        }
      })
      .finally(() => {
        setIsUpdatingPack(false);
      });
  };

  const onPackQueryEditorSubmit = (
    formData: IPackQueryFormData,
    editQuery: IScheduledQuery | undefined
  ) => {
    setIsUpdatingPack(true);
    const request = editQuery
      ? scheduledQueriesAPI.update(editQuery, formData)
      : scheduledQueriesAPI.create(formData);
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
        setIsUpdatingPack(false);
      });
    return false;
  };

  const onRemovePackQuerySubmit = () => {
    setIsUpdatingPack(true);
    const queryOrQueries =
      selectedPackQueryIds.length === 1 ? "query" : "queries";

    const promises = selectedPackQueryIds.map((id: number) => {
      return scheduledQueriesAPI.destroy(id);
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
        setIsUpdatingPack(false);
      });
  };

  return (
    <MainContent className={baseClass}>
      <>
        <div className={`${baseClass}__header-links`}>
          <BackLink text="Back to packs" path={PATHS.MANAGE_PACKS} />
        </div>
        {storedPack && storedPackQueries && (
          <EditPackForm
            className={`${baseClass}__pack-form`}
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
            isUpdatingPack={isUpdatingPack}
          />
        )}
        {showPackQueryEditorModal && queries && (
          <PackQueryEditorModal
            onCancel={togglePackQueryEditorModal}
            onPackQueryFormSubmit={onPackQueryEditorSubmit}
            allQueries={queries}
            editQuery={selectedPackQuery}
            packId={packId}
            isUpdatingPack={isUpdatingPack}
          />
        )}
        {showRemovePackQueryModal && queries && (
          <RemovePackQueryModal
            onCancel={toggleRemovePackQueryModal}
            onSubmit={onRemovePackQuerySubmit}
            selectedQuery={selectedPackQuery}
            selectedQueryIds={selectedPackQueryIds}
            isUpdatingPack={isUpdatingPack}
          />
        )}
      </>
    </MainContent>
  );
};

export default EditPacksPage;
