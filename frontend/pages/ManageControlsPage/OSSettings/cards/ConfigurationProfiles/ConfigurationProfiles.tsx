import React, { useCallback, useContext, useRef, useState } from "react";
import { useQuery } from "react-query";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import PageDescription from "components/PageDescription";
import Pagination from "components/Pagination";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import { getErrorReason } from "interfaces/errors";
import { IMdmAsset, IMdmProfile, isAnyMDMConfigured } from "interfaces/mdm";
import PATHS from "router/paths";
import mdmAPI, {
  IListAssetsResponse,
  IMdmProfilesResponse,
} from "services/entities/mdm";
import { getPathWithQueryParams } from "utilities/url";

import UploadList from "../../../../../components/UploadList";
import { IOSSettingsCommonProps } from "../../OSSettingsNavItems";

import AddAssetModal from "./components/AddAssetModal";
import AssetListItem from "./components/AssetListItem";
import ConfigProfilesFilters, {
  ConfigProfilesTypeFilter,
} from "./components/ConfigProfilesFilters";
import ConfigProfileStatusModal from "./components/ConfigProfileStatusModal";
import DeleteAssetModal from "./components/DeleteAssetModal";
import DeleteProfileModal from "./components/DeleteProfileModal/DeleteProfileModal";
import ProfileListItem from "./components/ProfileListItem";
import ResendConfigProfileModal from "./components/ResendConfigProfileModal";

const PROFILES_PER_PAGE = 10;

const baseClass = "configuration-profiles";

type IConfigProfilesListItem =
  | { key: string; kind: "profile"; profile: IMdmProfile }
  | { key: string; kind: "asset"; asset: IMdmAsset };

export type IConfigurationProfilesProps = IOSSettingsCommonProps & {
  currentPage?: number;
};

const ConfigurationProfiles = ({
  currentTeamId,
  router,
  currentPage = 0,
  onMutation,
}: IConfigurationProfilesProps) => {
  const {
    config,
    isPremiumTier,
    isGlobalAdmin,
    isGlobalTechnician,
    isTeamTechnician,
  } = useContext(AppContext);

  const isTechnician = isGlobalTechnician || isTeamTechnician;
  const canAddConfigurationProfile = !isTechnician;
  // The "Turn on" button links to /settings/integrations/mdm, which is
  // gated to global admins only (AuthGlobalAdminRoutes).
  const canTurnOnMdm = !!isGlobalAdmin;

  const mdmEnabled = isAnyMDMConfigured(config?.mdm);
  // Assets are only referenced by Apple declaration (DDM) profiles.
  const assetsEnabled = !!isPremiumTier && !!config?.mdm.enabled_and_configured;

  const [showDeleteProfileModal, setShowDeleteProfileModal] = useState(false);
  const [
    showConfigProfileStatusModal,
    setShowConfigProfileStatusModal,
  ] = useState(false);
  const [
    showResendConfigProfileModal,
    setShowResendConfigProfileModal,
  ] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [showAddAssetModal, setShowAddAssetModal] = useState(false);
  const [showDeleteAssetModal, setShowDeleteAssetModal] = useState(false);
  const [isDeletingAsset, setIsDeletingAsset] = useState(false);

  const [searchQuery, setSearchQuery] = useState("");
  const [labelIdFilter, setLabelIdFilter] = useState<number | undefined>();
  const [typeFilter, setTypeFilter] = useState<ConfigProfilesTypeFilter>("all");
  const hasActiveFilters =
    searchQuery !== "" || labelIdFilter !== undefined || typeFilter !== "all";
  // Assets have no label targets, so a label filter excludes them.
  const showAssets =
    assetsEnabled && typeFilter !== "profile" && labelIdFilter === undefined;
  const showProfiles = typeFilter !== "asset";

  const selectedProfile = useRef<IMdmProfile | null>(null);
  const selectedAsset = useRef<IMdmAsset | null>(null);
  const selectedStatusHostCount = useRef<number | null>(null);

  const {
    data: profilesData,
    isLoading: isLoadingProfiles,
    isError: isErrorProfiles,
    refetch: refetchProfiles,
  } = useQuery<IMdmProfilesResponse, unknown>(
    [
      {
        scope: "profiles",
        team_id: currentTeamId,
        page: currentPage,
        per_page: PROFILES_PER_PAGE,
        query: searchQuery,
        label_id: labelIdFilter,
      },
    ],
    () =>
      mdmAPI.getProfiles({
        fleet_id: currentTeamId,
        page: currentPage,
        per_page: PROFILES_PER_PAGE,
        query: searchQuery,
        label_id: labelIdFilter,
      }),
    {
      enabled: mdmEnabled && showProfiles,
      keepPreviousData: true,
      refetchOnWindowFocus: false,
    }
  );
  const profiles = profilesData?.profiles;
  const meta = profilesData?.meta;

  const {
    data: assets,
    isLoading: isLoadingAssets,
    refetch: refetchAssets,
  } = useQuery<IListAssetsResponse, unknown, IMdmAsset[]>(
    [{ scope: "assets", team_id: currentTeamId }],
    () => mdmAPI.getAssets({ fleet_id: currentTeamId }),
    {
      enabled: assetsEnabled,
      refetchOnWindowFocus: false,
      select: (res) => res.assets ?? [],
      // Assets are secondary here; don't hide the profiles if they fail.
      onError: (e) =>
        notify.error("Couldn't load assets. Please try again.", {
          response: e,
        }),
    }
  );

  const onCancelInfo = () => {
    selectedProfile.current = null;
    setShowConfigProfileStatusModal(false);
  };

  // Add and edit are full pages; keep the fleet so they come back here.
  const fleetQuery = { fleet_id: isPremiumTier ? currentTeamId : undefined };
  const onClickAdd = () => {
    router.push(
      getPathWithQueryParams(PATHS.CONTROLS_CUSTOM_SETTINGS_NEW, fleetQuery)
    );
  };

  const onCancelDelete = () => {
    selectedProfile.current = null;
    setShowDeleteProfileModal(false);
  };

  const onDeleteProfile = async (profileId: string) => {
    setIsDeleting(true);
    try {
      await mdmAPI.deleteProfile(profileId);
      refetchProfiles();
      onMutation();
      notify.success("Successfully deleted.");
    } catch (e) {
      const reason = getErrorReason(e, {
        reasonIncludes: "Policy automations",
      });
      if (reason === "") {
        notify.error("Couldn't delete. Please try again.", { response: e });
      } else {
        notify.error(reason, { response: e });
      }
    } finally {
      selectedProfile.current = null;
      setShowDeleteProfileModal(false);
    }
    setIsDeleting(false);
  };

  const onClickDeleteAsset = (asset: IMdmAsset) => {
    selectedAsset.current = asset;
    setShowDeleteAssetModal(true);
  };

  const onCancelDeleteAsset = () => {
    selectedAsset.current = null;
    setShowDeleteAssetModal(false);
  };

  const onDeleteAsset = async (assetUuid: string) => {
    setIsDeletingAsset(true);
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
      setIsDeletingAsset(false);
    }
  };

  // pagination controls
  const path = PATHS.CONTROLS_CUSTOM_SETTINGS;
  const queryString = isPremiumTier ? `?fleet_id=${currentTeamId}&` : "?";

  const onPrevPage = useCallback(() => {
    router.push(path.concat(`${queryString}page=${currentPage - 1}`));
  }, [router, path, currentPage, queryString]);

  const onNextPage = useCallback(() => {
    router.push(path.concat(`${queryString}page=${currentPage + 1}`));
  }, [router, path, currentPage, queryString]);

  // Filters change the result set, so go back to the first page.
  const resetToFirstPage = () => {
    if (currentPage !== 0) {
      router.replace(path.concat(`${queryString}page=0`));
    }
  };

  const onSearchQueryChange = (value: string) => {
    setSearchQuery(value.trim());
    resetToFirstPage();
  };

  const onTypeFilterChange = (type: ConfigProfilesTypeFilter) => {
    setTypeFilter(type);
    resetToFirstPage();
  };

  const onLabelIdChange = (labelId: number | undefined) => {
    setLabelIdFilter(labelId);
    resetToFirstPage();
  };

  const onClickInfo = (profile: IMdmProfile) => {
    selectedProfile.current = profile;
    setShowConfigProfileStatusModal(true);
  };

  const onClickEdit = (profile: IMdmProfile) => {
    router.push(
      getPathWithQueryParams(
        PATHS.CONTROLS_CUSTOM_SETTINGS_EDIT(profile.profile_uuid),
        fleetQuery
      )
    );
  };

  const onClickDelete = (profile: IMdmProfile) => {
    selectedProfile.current = profile;
    setShowDeleteProfileModal(true);
  };

  const renderProfileList = () => {
    if (
      (showProfiles && isLoadingProfiles) ||
      (assetsEnabled && isLoadingAssets)
    ) {
      return <Spinner />;
    }

    if (isErrorProfiles) {
      return <DataError />;
    }

    // Profiles are paginated and filtered server-side and assets aren't, so
    // assets are filtered here and listed once, on the first page.
    const filteredAssets =
      currentPage === 0 && showAssets
        ? (assets ?? []).filter((asset) =>
            asset.name.toLowerCase().includes(searchQuery.toLowerCase())
          )
        : [];
    const listItems: IConfigProfilesListItem[] = [
      ...filteredAssets.map(
        (asset): IConfigProfilesListItem => ({
          key: asset.asset_uuid,
          kind: "asset",
          asset,
        })
      ),
      ...(showProfiles ? profiles ?? [] : []).map(
        (profile): IConfigProfilesListItem => ({
          key: profile.profile_uuid,
          kind: "profile",
          profile,
        })
      ),
    ];

    if (!listItems.length && hasActiveFilters) {
      return (
        <EmptyState
          variant="list"
          header="No items match the current search criteria"
          info="Try a different search or filter."
        />
      );
    }

    if (!listItems.length) {
      return (
        <EmptyState
          variant="list"
          header="No configuration profiles"
          info={
            canAddConfigurationProfile
              ? "Add a configuration profile to enforce custom settings on your hosts."
              : "No configuration profiles have been added."
          }
          primaryButton={
            canAddConfigurationProfile ? (
              <GitOpsModeTooltipWrapper
                renderChildren={(disableChildren) => (
                  <Button disabled={disableChildren} onClick={onClickAdd}>
                    Add profile
                  </Button>
                )}
              />
            ) : undefined
          }
        />
      );
    }

    return (
      <>
        <UploadList
          keyAttribute="key"
          listItems={listItems}
          ListItemComponent={({ listItem }) =>
            listItem.kind === "asset" ? (
              <AssetListItem
                asset={listItem.asset}
                onClickDelete={onClickDeleteAsset}
                isTechnician={isTechnician}
              />
            ) : (
              <ProfileListItem
                isPremium={!!isPremiumTier}
                profile={listItem.profile}
                onClickInfo={onClickInfo}
                onClickEdit={onClickEdit}
                onClickDelete={onClickDelete}
                isTechnician={isTechnician}
              />
            )
          }
        />
        <Pagination
          disableNext={!showProfiles || !meta?.has_next_results}
          disablePrev={!meta?.has_previous_results}
          hidePagination={
            !showProfiles ||
            (!meta?.has_next_results && !meta?.has_previous_results)
          }
          onNextPage={onNextPage}
          onPrevPage={onPrevPage}
        />
      </>
    );
  };

  const profilesDescription = (
    <>
      {isTechnician
        ? "View configuration profiles."
        : "Create and upload configuration profiles to apply custom settings."}{" "}
      <CustomLink
        newTab
        text="Learn more"
        url="https://fleetdm.com/guides/custom-os-settings"
      />
    </>
  );

  // Hide the filters on an empty first page, where they'd filter nothing.
  const isEmptyUnfiltered =
    !hasActiveFilters &&
    currentPage === 0 &&
    !profiles?.length &&
    !(assetsEnabled && assets?.length);

  const showAddProfileButton = mdmEnabled && canAddConfigurationProfile;
  const showAdvancedButton = assetsEnabled && canAddConfigurationProfile;

  return (
    <div className={baseClass}>
      <SectionHeader title="Configuration profiles" alignLeftHeaderVertically />
      <div className={`${baseClass}__header`}>
        <PageDescription variant="right-panel" content={profilesDescription} />
        {(showAdvancedButton || showAddProfileButton) && (
          <GitOpsModeTooltipWrapper
            position="left"
            renderChildren={(disableChildren) => (
              <div className={`${baseClass}__header-actions`}>
                {showAdvancedButton && (
                  <Button
                    variant="secondary"
                    size="small"
                    onClick={() => setShowAddAssetModal(true)}
                    disabled={disableChildren}
                  >
                    Advanced
                  </Button>
                )}
                {showAddProfileButton && (
                  <Button
                    size="small"
                    onClick={onClickAdd}
                    disabled={disableChildren}
                    icon="plus"
                  >
                    Add profile
                  </Button>
                )}
              </div>
            )}
          />
        )}
      </div>
      {!mdmEnabled ? (
        <EmptyState
          variant="header-list"
          header="Additional configuration required"
          info="MDM must be turned on to add configuration profiles."
          primaryButton={
            canTurnOnMdm ? (
              <Button onClick={() => router.push(PATHS.ADMIN_INTEGRATIONS_MDM)}>
                Turn on
              </Button>
            ) : undefined
          }
        />
      ) : (
        <>
          {!isEmptyUnfiltered && (
            <ConfigProfilesFilters
              currentTeamId={currentTeamId}
              showTypeFilter={assetsEnabled}
              searchQuery={searchQuery}
              typeFilter={typeFilter}
              labelId={labelIdFilter}
              onSearchQueryChange={onSearchQueryChange}
              onTypeFilterChange={onTypeFilterChange}
              onLabelIdChange={onLabelIdChange}
            />
          )}
          {renderProfileList()}
        </>
      )}
      {showAddAssetModal && (
        <AddAssetModal
          currentTeamId={currentTeamId}
          onUpload={refetchAssets}
          closeModal={() => setShowAddAssetModal(false)}
        />
      )}
      {showDeleteAssetModal && selectedAsset.current && (
        <DeleteAssetModal
          assetUuid={selectedAsset.current.asset_uuid}
          onCancel={onCancelDeleteAsset}
          onDelete={onDeleteAsset}
          isDeleting={isDeletingAsset}
        />
      )}
      {showDeleteProfileModal && selectedProfile.current && (
        <DeleteProfileModal
          profileName={selectedProfile.current.name}
          profileId={selectedProfile.current.profile_uuid}
          platform={selectedProfile.current.platform}
          onCancel={onCancelDelete}
          onDelete={onDeleteProfile}
          isDeleting={isDeleting}
        />
      )}
      {showConfigProfileStatusModal && selectedProfile.current && (
        <ConfigProfileStatusModal
          teamId={currentTeamId}
          name={selectedProfile.current.name}
          uuid={selectedProfile.current.profile_uuid}
          payloadDisplayName={selectedProfile.current.payload_display_name}
          platform={selectedProfile.current.platform}
          onClickResend={(hostCount) => {
            selectedStatusHostCount.current = hostCount;
            setShowConfigProfileStatusModal(false);
            setShowResendConfigProfileModal(true);
          }}
          onExit={onCancelInfo}
        />
      )}
      {showResendConfigProfileModal &&
        selectedProfile.current &&
        selectedStatusHostCount.current && (
          <ResendConfigProfileModal
            name={selectedProfile.current.name}
            uuid={selectedProfile.current.profile_uuid}
            count={selectedStatusHostCount.current}
            onExit={() => {
              selectedStatusHostCount.current = null;
              setShowResendConfigProfileModal(false);
              setShowConfigProfileStatusModal(true);
            }}
          />
        )}
    </div>
  );
};

export default ConfigurationProfiles;
