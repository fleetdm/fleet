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
import Card from "components/Card/Card";
import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import UploadList from "components/UploadList";

import UploadListHeading from "../../../../../components/UploadListHeading";
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
    isAnyTeamAdmin,
    isGlobalTechnician,
    isTeamTechnician,
  } = useContext(AppContext);

  const isTechnician = isGlobalTechnician || isTeamTechnician;
  // Mirrors the route guard on /settings/integrations/mdm/apple (AuthAnyAdminRoutes).
  const canTurnOnMdm = isGlobalAdmin || isAnyTeamAdmin;
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
          info={
            canTurnOnMdm
              ? "Supported on macOS, iOS, and iPadOS."
              : "To manage assets, ask your admin to turn on Apple MDM."
          }
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
      if (isTechnician) {
        return <Card className="empty-assets">No assets have been added.</Card>;
      }
      return (
        <EmptyState
          variant="header-list"
          header="No assets"
          info="Add an asset to make it available for reference in Apple DDM declarations."
          primaryButton={
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
          }
        />
      );
    }

    return (
      <UploadList
        keyAttribute="asset_uuid"
        listItems={assets}
        HeadingComponent={() => (
          <UploadListHeading
            onClickAdd={
              isTechnician ? undefined : () => setShowAddAssetModal(true)
            }
            entityName="Assets"
            createEntityText="Add asset"
          />
        )}
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

  return (
    <div className={baseClass}>
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
