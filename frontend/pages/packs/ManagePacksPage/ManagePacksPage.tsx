import React, { useState, useCallback, useContext } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useQuery } from "react-query";

import { IPack } from "interfaces/pack";
import { IError } from "interfaces/errors";

import { AppContext } from "context/app";
import packsAPI from "services/entities/packs";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import PATHS from "router/paths";
// @ts-ignore
import deepDifference from "utilities/deep_difference";

import Button from "components/buttons/Button";
import TableDataError from "components/TableDataError";
import PacksListWrapper from "./components/PacksListWrapper";
import RemovePackModal from "./components/RemovePackModal";

const baseClass = "manage-packs-page";

interface IManagePacksPageProps {
  router: any;
}

interface IPacksResponse {
  packs: IPack[];
}

const renderTable = (
  onRemovePackClick: React.MouseEventHandler<HTMLButtonElement>,
  onEnablePackClick: React.MouseEventHandler<HTMLButtonElement>,
  onDisablePackClick: React.MouseEventHandler<HTMLButtonElement>,
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

  const dispatch = useDispatch();

  const onCreatePackClick = () => router.push(PATHS.NEW_PACK);

  const [selectedPackIds, setSelectedPackIds] = useState<number[]>([]);
  const [showRemovePackModal, setShowRemovePackModal] = useState<boolean>(
    false
  );

  const {
    data: packs,
    error: packsError,
    isLoading: isLoadingPacks,
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

  const onRemovePackClick = (selectedTablePackIds: any) => {
    toggleRemovePackModal();
    setSelectedPackIds(selectedTablePackIds);
  };

  const onRemovePackSubmit = useCallback(() => {
    const packOrPacks = selectedPackIds.length === 1 ? "pack" : "packs";

    const promises = selectedPackIds.map((id: number) => {
      packsAPI.destroy(id);
      return null;
    });

    return Promise.all(promises)
      .then(() => {
        dispatch(
          renderFlash("success", `Successfully deleted ${packOrPacks}.`)
        );
        toggleRemovePackModal();
      })
      .catch(() => {
        dispatch(
          renderFlash(
            "error",
            `Unable to remove ${packOrPacks}. Please try again.`
          )
        );
      })
      .finally(() => {
        refetchPacks();
        toggleRemovePackModal();
      });
  }, [dispatch, refetchPacks, selectedPackIds, toggleRemovePackModal]);

  const onEnableDisablePackSubmit = useCallback(
    (selectedTablePackIds: any, disablePack: boolean) => {
      const packOrPacks = selectedPackIds.length === 1 ? "pack" : "packs";
      const enableOrDisable = disablePack ? "disabled" : "enabled";

      const promises = selectedTablePackIds.map((id: number) => {
        packsAPI.update(id, { disabled: disablePack });
      });

      return Promise.all(promises)
        .then(() => {
          dispatch(
            renderFlash(
              "success",
              `Successfully ${enableOrDisable} selected ${packOrPacks}.`
            )
          );
        })
        .catch(() => {
          dispatch(
            renderFlash(
              "error",
              `Unable to ${enableOrDisable} selected ${packOrPacks}. Please try again.`
            )
          );
        })
        .finally(() => {
          refetchPacks();
        });
    },
    [dispatch, refetchPacks, selectedPackIds]
  );

  const onEnablePackClick = (selectedTablePackIds: any) => {
    setSelectedPackIds(selectedTablePackIds);
    onEnableDisablePackSubmit(selectedTablePackIds, false);
  };

  const onDisablePackClick = (selectedTablePackIds: any) => {
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
          {!isLoadingPacks &&
            renderTable(
              onRemovePackClick,
              onEnablePackClick,
              onDisablePackClick,
              onCreatePackClick,
              packs,
              packsError,
              isLoadingPacks
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
