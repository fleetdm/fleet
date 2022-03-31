import React, { useState, useCallback, useContext } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";

import { IPack } from "interfaces/pack";
import { IError } from "interfaces/errors";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import packsAPI from "services/entities/packs";
import PATHS from "router/paths";

// @ts-ignore
import Button from "components/buttons/Button";
import TableDataError from "components/TableDataError";
import Spinner from "components/Spinner";
import PacksListWrapper from "./components/PacksListWrapper";
import RemovePackModal from "./components/RemovePackModal";

const baseClass = "manage-packs-page";

interface IManagePacksPageProps {
  router: InjectedRouter; // v3
}

interface IPacksResponse {
  packs: IPack[];
}

const renderTable = (
  onRemovePackClick: (selectedTablePackIds: number[]) => void,
  onEnablePackClick: (selectedTablePackIds: number[]) => void,
  onDisablePackClick: (selectedTablePackIds: number[]) => void,
  onCreatePackClick: React.MouseEventHandler<HTMLButtonElement>,
  packs: IPack[] | undefined,
  packsError: IError | null,
  isLoadingPacks: boolean
): JSX.Element => {
  if (packsError) {
    return <TableDataError />;
  }

  const isTableDataLoading = isLoadingPacks || packs === null;

  return (
    <PacksListWrapper
      onRemovePackClick={onRemovePackClick}
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
  const [showRemovePackModal, setShowRemovePackModal] = useState<boolean>(
    false
  );

  const {
    data: packs,
    error: packsError,
    isFetching: isLoadingPacks,
    refetch: refetchPacks,
  } = useQuery<IPacksResponse, IError, IPack[]>(
    "packs",
    () => packsAPI.loadAll(),
    {
      // refetchOnMount: false,
      // refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      select: (data: IPacksResponse) => data.packs,
    }
  );

  const toggleRemovePackModal = useCallback(() => {
    setShowRemovePackModal(!showRemovePackModal);
  }, [showRemovePackModal, setShowRemovePackModal]);

  const onRemovePackClick = (selectedTablePackIds: number[]) => {
    toggleRemovePackModal();
    setSelectedPackIds(selectedTablePackIds);
  };

  const onRemovePackSubmit = useCallback(() => {
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
          `Unable to remove ${packOrPacks}. Please try again.`
        );
      })
      .finally(() => {
        refetchPacks();
        toggleRemovePackModal();
      });
  }, [refetchPacks, selectedPackIds, toggleRemovePackModal]);

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
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
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
                variant="brand"
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
              onRemovePackClick,
              onEnablePackClick,
              onDisablePackClick,
              onCreatePackClick,
              packs,
              packsError,
              isLoadingPacks
            )
          )}
        </div>
        {showRemovePackModal && (
          <RemovePackModal
            onCancel={toggleRemovePackModal}
            onSubmit={onRemovePackSubmit}
          />
        )}
      </div>
    </div>
  );
};

export default ManagePacksPage;
