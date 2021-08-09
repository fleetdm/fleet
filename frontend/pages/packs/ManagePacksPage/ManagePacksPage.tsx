import React, { useState, useCallback, useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";

import { push } from "react-router-redux";
import pack, { IPack } from "interfaces/pack";
import { IUser } from "interfaces/user";

// @ts-ignore
import packActions from "redux/nodes/entities/packs/actions";
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";

import paths from "router/paths";
import permissionUtils from "utilities/permissions";
// @ts-ignore
import deepDifference from "utilities/deep_difference";

import Button from "components/buttons/Button";
import PacksListError from "./components/PacksListError";
import PacksListWrapper from "./components/PacksListWrapper";
import RemovePackModal from "./components/RemovePackModal";

const baseClass = "manage-packs-page";
interface IRootState {
  auth: {
    user: IUser;
  };
  entities: {
    packs: {
      isLoading: boolean;
      data: IPack[];
      errors: any;
    };
  };
}

const renderTable = (
  onRemovePackClick: React.MouseEventHandler<HTMLButtonElement>,
  onEnablePackClick: React.MouseEventHandler<HTMLButtonElement>,
  onDisablePackClick: React.MouseEventHandler<HTMLButtonElement>,
  packsList: IPack[],
  packsErrors: any
): JSX.Element => {
  if (Object.keys(packsErrors).length > 0) {
    return <PacksListError />;
  }

  return (
    <PacksListWrapper
      onRemovePackClick={onRemovePackClick}
      onEnablePackClick={onEnablePackClick}
      onDisablePackClick={onDisablePackClick}
      packsList={packsList}
    />
  );
};

const ManagePacksPage = (): JSX.Element => {
  const currentUser = useSelector((state: IRootState) => state.auth.user);
  const isOnlyObserver = permissionUtils.isOnlyObserver(currentUser);

  const dispatch = useDispatch();
  const { NEW_PACK } = paths;
  const onCreatePackClick = () => dispatch(push(NEW_PACK));

  useEffect(() => {
    dispatch(packActions.loadAll());
  }, [dispatch]);

  const packs = useSelector((state: IRootState) => state.entities.packs);
  const packsList = Object.values(packs.data);
  const packsErrors = packs.errors;

  const [selectedPackIds, setSelectedPackIds] = useState<number[]>([]);
  const [showRemovePackModal, setShowRemovePackModal] = useState<boolean>(
    false
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
      return dispatch(packActions.destroy({ id }));
    });

    return Promise.all(promises)
      .then(() => {
        dispatch(
          renderFlash("success", `Successfully deleted ${packOrPacks}.`)
        );
        toggleRemovePackModal();
        dispatch(packActions.loadAll());
      })
      .catch(() => {
        dispatch(
          renderFlash(
            "error",
            `Unable to remove ${packOrPacks}. Please try again.`
          )
        );
        toggleRemovePackModal();
      });
  }, [dispatch, selectedPackIds, toggleRemovePackModal]);

  const onEnableDisablePackSubmit = useCallback(
    (selectedTablePackIds: any, disablePack: boolean) => {
      const packOrPacks = selectedPackIds.length === 1 ? "pack" : "packs";
      const enableOrDisable = disablePack ? "disabled" : "enabled";

      const promises = selectedTablePackIds.map((id: number) => {
        return dispatch(packActions.update({ id }, { disabled: disablePack }));
      });

      return Promise.all(promises)
        .then(() => {
          dispatch(
            renderFlash(
              "success",
              `Successfully ${enableOrDisable} selected ${packOrPacks}.`
            )
          );
          dispatch(packActions.loadAll());
        })
        .catch(() => {
          dispatch(
            renderFlash(
              "error",
              `Unable to ${enableOrDisable} selected ${packOrPacks}. Please try again.`
            )
          );
        });
    },
    [dispatch, selectedPackIds]
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
          {!isOnlyObserver && (
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
          {!packs.isLoading &&
            renderTable(
              onRemovePackClick,
              onEnablePackClick,
              onDisablePackClick,
              packsList,
              packsErrors
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
