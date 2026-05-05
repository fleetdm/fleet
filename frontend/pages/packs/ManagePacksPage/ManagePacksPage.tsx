import React, { useState, useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";

import { IPack, IStoredPacksResponse } from "interfaces/pack";
import { IFleetApiError } from "interfaces/errors";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import packsAPI from "services/entities/packs";
import PATHS from "router/paths";

// @ts-ignore
import Button from "components/buttons/Button";
import TableDataError from "components/DataError";
import Spinner from "components/Spinner";
import MainContent from "components/MainContent";
import PacksTable from "./components/PacksTable";
import DeletePackModal from "./components/DeletePackModal";

const baseClass = "manage-packs-page";

interface IManagePacksPageProps {
  router: InjectedRouter; // v3
}

const renderTable = (
  onDeletePackClick: (selectedTablePackIds: number[]) => void,
  onEnablePackClick: (selectedTablePackIds: number[]) => void,
  onDisablePackClick: (selectedTablePackIds: number[]) => void,
  onCreatePackClick: React.MouseEventHandler<HTMLButtonElement>,
  packs: IPack[] | undefined,
  packsError: IFleetApiError | null,
  isLoadingPacks: boolean
): JSX.Element => {
  if (packsError) {
    return <TableDataError />;
  }

  const isTableDataLoading = isLoadingPacks || packs === null;

  return (
    <PacksTable
      onDeletePackClick={onDeletePackClick}
      onEnablePackClick={onEnablePackClick}
      onDisablePackClick={onDisablePackClick}
      onCreatePackClick={onCreatePackClick}
      packs={packs}
      isLoading={isTableDataLoading}
    />
  );
};

const ManagePacksPage = ({ router }: IManagePacksPageProps): JSX.Element => {
  const { isOnlyObserver } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const onCreatePackClick = () => router.push(PATHS.NEW_PACK);

  const [selectedPackIds, setSelectedPackIds] = useState<number[]>([]);
  const [showDeletePackModal, setShowDeletePackModal] = useState(false);
  const [isUpdatingPack, setIsUpdatingPack] = useState(false);

  const {
    data: packs,
    error: packsError,
    isFetching: isLoadingPacks,
    refetch: refetchPacks,
  } = useQuery<IStoredPacksResponse, IFleetApiError, IPack[]>(
    "packs",
    () => packsAPI.loadAll(),
    {
      // refetchOnMount: false,
      // refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      select: (data: IStoredPacksResponse) => data.packs,
    }
  );

  const toggleDeletePackModal = useCallback(() => {
    setShowDeletePackModal(!showDeletePackModal);
  }, [showDeletePackModal, setShowDeletePackModal]);

  const onDeletePackClick = (selectedTablePackIds: number[]) => {
    toggleDeletePackModal();
    setSelectedPackIds(selectedTablePackIds);
  };

  const onDeletePackSubmit = useCallback(() => {
    setIsUpdatingPack(true);
    const packOrPacks = selectedPackIds.length === 1 ? "pack" : "packs";

    const promises = selectedPackIds.map((id: number) => {
      return packsAPI.destroy(id);
    });

    return Promise.all(promises)
      .then(() => {
        renderFlash("success", `Successfully deleted ${packOrPacks}.`);
      })
      .catch(() => {
        renderFlash(
          "error",
          `Unable to delete ${packOrPacks}. Please try again.`
        );
      })
      .finally(() => {
        refetchPacks();
        toggleDeletePackModal();
        setIsUpdatingPack(false);
      });
  }, [refetchPacks, selectedPackIds, toggleDeletePackModal]);

  const onEnableDisablePackSubmit = useCallback(
    (selectedTablePackIds: number[], disablePack: boolean) => {
      const packOrPacks = selectedPackIds.length === 1 ? "pack" : "packs";
      const enableOrDisable = disablePack ? "disabled" : "enabled";

      const promises = selectedTablePackIds.map((id: number) => {
        return packsAPI.update(id, { disabled: disablePack });
      });

      return Promise.all(promises)
        .then(() => {
          renderFlash(
            "success",
            `Successfully ${enableOrDisable} selected ${packOrPacks}.`
          );
        })
        .catch(() => {
          renderFlash(
            "error",
            `Unable to ${enableOrDisable} selected ${packOrPacks}. Please try again.`
          );
        })
        .finally(() => {
          refetchPacks();
        });
    },
    [refetchPacks, selectedPackIds]
  );

  const onEnablePackClick = (selectedTablePackIds: number[]) => {
    setSelectedPackIds(selectedTablePackIds);
    onEnableDisablePackSubmit(selectedTablePackIds, false);
  };

  const onDisablePackClick = (selectedTablePackIds: number[]) => {
    setSelectedPackIds(selectedTablePackIds);
    onEnableDisablePackSubmit(selectedTablePackIds, true);
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <h1 className={`${baseClass}__title`}>
                <span>Packs</span>
              </h1>
              <div className={`${baseClass}__description`}>
                <p>
                  Manage query packs to schedule recurring queries for your
                  hosts.
                </p>
              </div>
            </div>
          </div>
          {!isOnlyObserver && packs && packs.length > 0 && (
            <div className={`${baseClass}__action-button-container`}>
              <Button
                className={`${baseClass}__create-button`}
                onClick={onCreatePackClick}
              >
                Create new pack
              </Button>
            </div>
          )}
        </div>
        <div>
          {isLoadingPacks ? (
            <Spinner />
          ) : (
            renderTable(
              onDeletePackClick,
              onEnablePackClick,
              onDisablePackClick,
              onCreatePackClick,
              packs,
              packsError,
              isLoadingPacks
            )
          )}
        </div>
        {showDeletePackModal && (
          <DeletePackModal
            onCancel={toggleDeletePackModal}
            onSubmit={onDeletePackSubmit}
            isUpdatingPack={isUpdatingPack}
          />
        )}
      </div>
    </MainContent>
  );
};

export default ManagePacksPage;
