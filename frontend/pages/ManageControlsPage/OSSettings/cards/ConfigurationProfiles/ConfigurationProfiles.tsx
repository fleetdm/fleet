import React, { useCallback, useContext, useRef, useState } from "react";
import { useQuery } from "react-query";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import PageDescription from "components/PageDescription";
import Pagination from "components/Pagination";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import { getErrorReason } from "interfaces/errors";
import { IMdmProfile, isAnyMDMConfigured } from "interfaces/mdm";
import PATHS from "router/paths";
import mdmAPI, { IMdmProfilesResponse } from "services/entities/mdm";
import { getPathWithQueryParams } from "utilities/url";

import UploadList from "../../../../../components/UploadList";
import { IOSSettingsCommonProps } from "../../OSSettingsNavItems";

import AssetsTab from "./components/AssetsTab";
import ConfigProfileStatusModal from "./components/ConfigProfileStatusModal";
import DeleteProfileModal from "./components/DeleteProfileModal/DeleteProfileModal";
import ProfileListItem from "./components/ProfileListItem";
import ResendConfigProfileModal from "./components/ResendConfigProfileModal";

const PROFILES_PER_PAGE = 10;

const baseClass = "configuration-profiles";

export type ConfigProfilesTab = "profiles" | "assets";

const TABS_BY_INDEX: ConfigProfilesTab[] = ["profiles", "assets"];

export type IConfigurationProfilesProps = IOSSettingsCommonProps & {
  currentPage?: number;
  /** Which secondary tab is active, derived from the route section. */
  activeTab?: ConfigProfilesTab;
};

const ConfigurationProfiles = ({
  currentTeamId,
  router,
  currentPage = 0,
  activeTab = "profiles",
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

  const selectedProfile = useRef<IMdmProfile | null>(null);
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
      },
    ],
    () =>
      mdmAPI.getProfiles({
        fleet_id: currentTeamId,
        page: currentPage,
        per_page: PROFILES_PER_PAGE,
      }),
    {
      enabled: mdmEnabled,
      refetchOnWindowFocus: false,
    }
  );
  const profiles = profilesData?.profiles;
  const meta = profilesData?.meta;

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

  // pagination controls
  const path = PATHS.CONTROLS_CUSTOM_SETTINGS;
  const queryString = isPremiumTier ? `?fleet_id=${currentTeamId}&` : "?";

  const onPrevPage = useCallback(() => {
    router.push(path.concat(`${queryString}page=${currentPage - 1}`));
  }, [router, path, currentPage, queryString]);

  const onNextPage = useCallback(() => {
    router.push(path.concat(`${queryString}page=${currentPage + 1}`));
  }, [router, path, currentPage, queryString]);

  const handleTabChange = (index: number) => {
    const tabPath =
      TABS_BY_INDEX[index] === "assets"
        ? PATHS.CONTROLS_ASSETS
        : PATHS.CONTROLS_CUSTOM_SETTINGS;
    router.push(
      getPathWithQueryParams(tabPath, {
        fleet_id: isPremiumTier ? currentTeamId : undefined,
      })
    );
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
    if (isLoadingProfiles) {
      return <Spinner />;
    }

    if (isErrorProfiles) {
      return <DataError />;
    }

    if (!profiles?.length) {
      return (
        <EmptyState
          variant="header-list"
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
          keyAttribute="profile_uuid"
          listItems={profiles}
          ListItemComponent={({ listItem }) => (
            <ProfileListItem
              isPremium={!!isPremiumTier}
              profile={listItem}
              onClickInfo={onClickInfo}
              onClickEdit={onClickEdit}
              onClickDelete={onClickDelete}
              isTechnician={isTechnician}
            />
          )}
        />
        <Pagination
          disableNext={!meta?.has_next_results}
          disablePrev={!meta?.has_previous_results}
          hidePagination={
            !meta?.has_next_results && !meta?.has_previous_results
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

  const showAddProfileButton = mdmEnabled && canAddConfigurationProfile;

  return (
    <div className={baseClass}>
      <SectionHeader title="Configuration profiles" alignLeftHeaderVertically />
      <TabNav secondary>
        <Tabs
          selectedIndex={TABS_BY_INDEX.indexOf(activeTab)}
          onSelect={handleTabChange}
        >
          <TabList>
            <Tab>
              <TabText>Profiles</TabText>
            </Tab>
            <Tab>
              <TabText>Assets</TabText>
            </Tab>
          </TabList>
          <TabPanel>
            <div className="profiles-tab">
              <div className="profiles-tab__tab-header">
                <PageDescription
                  variant="right-panel"
                  content={profilesDescription}
                />
                {showAddProfileButton && (
                  <GitOpsModeTooltipWrapper
                    position="left"
                    renderChildren={(disableChildren) => (
                      <Button
                        variant="secondary"
                        size="small"
                        onClick={onClickAdd}
                        disabled={disableChildren}
                        icon="plus"
                      >
                        Add profile
                      </Button>
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
                      <Button
                        onClick={() =>
                          router.push(PATHS.ADMIN_INTEGRATIONS_MDM)
                        }
                      >
                        Turn on
                      </Button>
                    ) : undefined
                  }
                />
              ) : (
                renderProfileList()
              )}
            </div>
          </TabPanel>
          <TabPanel>
            <AssetsTab currentTeamId={currentTeamId} router={router} />
          </TabPanel>
        </Tabs>
      </TabNav>
      {showDeleteProfileModal && selectedProfile.current && (
        <DeleteProfileModal
          profileName={selectedProfile.current.name}
          profileId={selectedProfile.current.profile_uuid}
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
