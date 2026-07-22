import React, { useCallback, useContext, useRef, useState } from "react";

import { useQuery } from "react-query";
import { Tab, TabList, TabPanel, Tabs } from "react-tabs";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import { AppContext } from "context/app";
import { notify } from "components/ToastNotification";

import { IMdmProfile } from "interfaces/mdm";

import mdmAPI, { IMdmProfilesResponse } from "services/entities/mdm";

import Card from "components/Card/Card";
import CustomLink from "components/CustomLink";
import SectionHeader from "components/SectionHeader";
import PageDescription from "components/PageDescription";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
import Button from "components/buttons/Button";
import TabNav from "components/TabNav";
import TabText from "components/TabText";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import Pagination from "components/Pagination";

import UploadList from "../../../../../components/UploadList";

import AddProfileModal from "./components/ProfileUploader/components/AddProfileModal";
import DeleteProfileModal from "./components/DeleteProfileModal/DeleteProfileModal";
import ProfileLabelsModal from "./components/ProfileLabelsModal/ProfileLabelsModal";
import ProfileListItem from "./components/ProfileListItem";
import UploadListHeading from "../../../components/UploadListHeading";
import ConfigProfileStatusModal from "./components/ConfigProfileStatusModal";
import ResendConfigProfileModal from "./components/ResendConfigProfileModal";
import AssetsTab from "./components/AssetsTab";
import { IOSSettingsCommonProps } from "../../OSSettingsNavItems";

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
  // The "Turn on" button links to /settings/integrations/mdm, which is
  // gated to global admins only (AuthGlobalAdminRoutes).
  const canTurnOnMdm = !!isGlobalAdmin;

  const mdmEnabled =
    config?.mdm.enabled_and_configured ||
    config?.mdm.windows_enabled_and_configured ||
    config?.mdm.android_enabled_and_configured;

  const [showAddProfileModal, setShowAddProfileModal] = useState(false);
  const [
    profileLabelsModalData,
    setProfileLabelsModalData,
  ] = useState<IMdmProfile | null>(null);
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

  const onUploadProfile = () => {
    refetchProfiles();
    onMutation();
  };

  const onCancelInfo = () => {
    selectedProfile.current = null;
    setShowConfigProfileStatusModal(false);
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
      notify.error("Couldn't delete. Please try again.", { response: e });
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
      if (isTechnician) {
        return (
          <Card className="empty-profiles">
            No configuration profiles have been added.
          </Card>
        );
      }
      return (
        <EmptyState
          variant="header-list"
          header="No configuration profiles"
          info="Add a configuration profile to enforce custom settings on your hosts."
          primaryButton={
            <GitOpsModeTooltipWrapper
              renderChildren={(disableChildren) => (
                <Button
                  disabled={disableChildren}
                  onClick={() => setShowAddProfileModal(true)}
                >
                  Add profile
                </Button>
              )}
            />
          }
        />
      );
    }

    return (
      <>
        <UploadList
          keyAttribute="profile_uuid"
          listItems={profiles}
          HeadingComponent={() => (
            <UploadListHeading
              onClickAdd={
                isTechnician ? undefined : () => setShowAddProfileModal(true)
              }
              entityName="Configuration profile"
              createEntityText="Add profile"
            />
          )}
          ListItemComponent={({ listItem }) => (
            <ProfileListItem
              isPremium={!!isPremiumTier}
              profile={listItem}
              setProfileLabelsModalData={setProfileLabelsModalData}
              onClickInfo={onClickInfo}
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

  const hasLabels =
    !!profileLabelsModalData?.labels_include_all?.length ||
    !!profileLabelsModalData?.labels_include_any?.length ||
    !!profileLabelsModalData?.labels_exclude_any?.length;

  const pageDescription =
    activeTab === "assets" ? (
      "Manage assets that provide data or credentials referenced by DDM declarations."
    ) : (
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

  return (
    <div className={baseClass}>
      <SectionHeader title="Configuration profiles" alignLeftHeaderVertically />
      <PageDescription variant="right-panel" content={pageDescription} />
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
            {!mdmEnabled ? (
              <EmptyState
                variant="header-list"
                header="Additional configuration required"
                info="MDM must be turned on to add configuration profiles."
                primaryButton={
                  canTurnOnMdm ? (
                    <Button
                      onClick={() => router.push(PATHS.ADMIN_INTEGRATIONS_MDM)}
                    >
                      Turn on
                    </Button>
                  ) : undefined
                }
              />
            ) : (
              renderProfileList()
            )}
          </TabPanel>
          <TabPanel>
            <AssetsTab currentTeamId={currentTeamId} router={router} />
          </TabPanel>
        </Tabs>
      </TabNav>
      {showAddProfileModal && (
        <AddProfileModal
          currentTeamId={currentTeamId}
          isPremiumTier={!!isPremiumTier}
          onUpload={onUploadProfile}
          setShowModal={setShowAddProfileModal}
        />
      )}
      {showDeleteProfileModal && selectedProfile.current && (
        <DeleteProfileModal
          profileName={selectedProfile.current.name}
          profileId={selectedProfile.current.profile_uuid}
          onCancel={onCancelDelete}
          onDelete={onDeleteProfile}
          isDeleting={isDeleting}
        />
      )}
      {isPremiumTier && hasLabels && (
        <ProfileLabelsModal
          profile={profileLabelsModalData}
          setModalData={setProfileLabelsModalData}
        />
      )}
      {showConfigProfileStatusModal && selectedProfile.current && (
        <ConfigProfileStatusModal
          teamId={currentTeamId}
          name={selectedProfile.current.name}
          uuid={selectedProfile.current.profile_uuid}
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
