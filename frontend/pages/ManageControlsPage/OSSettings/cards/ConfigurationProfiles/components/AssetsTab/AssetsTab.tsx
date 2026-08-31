import React, { useContext, useRef, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { notify } from "components/ToastNotification";

import { getErrorReason } from "interfaces/errors";
import { IMdmAsset } from "interfaces/mdm";
import mdmAPI, { IListAssetsResponse } from "services/entities/mdm";

import Button from "components/buttons/Button";
import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import PageDescription from "components/PageDescription";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import UploadList from "components/UploadList";

import AssetListItem from "../AssetListItem";
import AddAssetModal from "../AddAssetModal";
import DeleteAssetModal from "../DeleteAssetModal";

const baseClass = "assets-tab";

interface IAssetsTabProps {
  currentTeamId: number;
  router: InjectedRouter;
}

const AssetsTab = ({ currentTeamId, router }: IAssetsTabProps) => {
  const {
    config,
    isPremiumTier,
    isGlobalAdmin,
    isGlobalTechnician,
    isTeamTechnician,
  } = useContext(AppContext);

  const isTechnician = isGlobalTechnician || isTeamTechnician;
  const canAddAsset = !isTechnician;
  // Team admins can reach /settings/integrations/mdm/apple, but only global
  // admins can actually turn on Apple MDM there.
  const canTurnOnMdm = !!isGlobalAdmin;
  const mdmAppleEnabled = !!config?.mdm.enabled_and_configured;

  const [showAddAssetModal, setShowAddAssetModal] = useState(false);
  const [showDeleteAssetModal, setShowDeleteAssetModal] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const selectedAsset = useRef<IMdmAsset | null>(null);

  const {
    data: assets,
    isLoading: isLoadingAssets,
    isError: isErrorAssets,
    refetch: refetchAssets,
  } = useQuery<IListAssetsResponse, unknown, IMdmAsset[]>(
    [{ scope: "assets", team_id: currentTeamId }],
    () => mdmAPI.getAssets({ fleet_id: currentTeamId }),
    {
      enabled: isPremiumTier && mdmAppleEnabled,
      refetchOnWindowFocus: false,
      select: (res) => res.assets ?? [],
    }
  );

  const onAddAsset = () => {
    refetchAssets();
  };

  const onClickDelete = (asset: IMdmAsset) => {
    selectedAsset.current = asset;
    setShowDeleteAssetModal(true);
  };

  const onCancelDelete = () => {
    selectedAsset.current = null;
    setShowDeleteAssetModal(false);
  };

  const onDeleteAsset = async (assetUuid: string) => {
    setIsDeleting(true);
    try {
      await mdmAPI.deleteAsset(assetUuid);
      refetchAssets();
      notify.success("Successfully deleted.");
    } catch (e) {
      notify.error(getErrorReason(e) || "Couldn't delete. Please try again.", {
        response: e,
      });
    } finally {
      selectedAsset.current = null;
      setShowDeleteAssetModal(false);
      setIsDeleting(false);
    }
  };

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }

    if (!mdmAppleEnabled) {
      return (
        <EmptyState
          variant="header-list"
          header="Manage assets"
          info="Supported on macOS, iOS, and iPadOS."
          primaryButton={
            canTurnOnMdm ? (
              <Button
                onClick={() => router.push(PATHS.ADMIN_INTEGRATIONS_MDM_APPLE)}
              >
                Turn on Apple MDM
              </Button>
            ) : undefined
          }
        />
      );
    }

    if (isLoadingAssets) {
      return <Spinner />;
    }

    if (isErrorAssets) {
      return <DataError />;
    }

    if (!assets?.length) {
      return (
        <EmptyState
          variant="header-list"
          header="No assets"
          info={
            canAddAsset
              ? "Add an asset to make it available for reference in Apple DDM declarations."
              : "No assets have been added."
          }
          primaryButton={
            canAddAsset ? (
              <GitOpsModeTooltipWrapper
                renderChildren={(disableChildren) => (
                  <Button
                    disabled={disableChildren}
                    onClick={() => setShowAddAssetModal(true)}
                  >
                    Add asset
                  </Button>
                )}
              />
            ) : undefined
          }
        />
      );
    }

    return (
      <UploadList
        keyAttribute="asset_uuid"
        listItems={assets}
        ListItemComponent={({ listItem }) => (
          <AssetListItem
            asset={listItem}
            onClickDelete={onClickDelete}
            isTechnician={isTechnician}
          />
        )}
      />
    );
  };

  const showAddAssetButton = isPremiumTier && mdmAppleEnabled && canAddAsset;

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__tab-header`}>
        <PageDescription
          variant="right-panel"
          content="Manage assets that provide data or credentials referenced by DDM declarations."
        />
        {showAddAssetButton && (
          <GitOpsModeTooltipWrapper
            position="left"
            renderChildren={(disableChildren) => (
              <Button
                variant="secondary"
                size="small"
                onClick={() => setShowAddAssetModal(true)}
                disabled={disableChildren}
                icon="plus"
              >
                Add asset
              </Button>
            )}
          />
        )}
      </div>
      {renderContent()}
      {showAddAssetModal && (
        <AddAssetModal
          currentTeamId={currentTeamId}
          onUpload={onAddAsset}
          closeModal={() => setShowAddAssetModal(false)}
        />
      )}
      {showDeleteAssetModal && selectedAsset.current && (
        <DeleteAssetModal
          assetUuid={selectedAsset.current.asset_uuid}
          onCancel={onCancelDelete}
          onDelete={onDeleteAsset}
          isDeleting={isDeleting}
        />
      )}
    </div>
  );
};

export default AssetsTab;
