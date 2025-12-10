import CustomLink from "components/CustomLink";
import PageDescription from "components/PageDescription";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SectionHeader from "components/SectionHeader";
import { AppContext } from "context/app";
import React, { useContext } from "react";
import { useQuery } from "react-query";
import { DEFAULT_USE_QUERY_OPTIONS, LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import mdmAPI, { IMdmCertsResponse } from "services/entities/mdm";

const baseClass = "certificates";

interface ICertificates {}

const Certificates = ({}: ICertificates) => {
  const { config, isPremiumTier } = useContext(AppContext);

  const {
    data: certs,
    isLoading: isLoadingCerts,
    isError: isErrorCerts,
    refetch: refetchCerts,
  } = useQuery<IMdmCertsResponse, unknown>(
    [
      {
        scope: "certificates",
        team_id: currentTeamId,
        page: currentPage,
        per_page: PROFILES_PER_PAGE,
      },
    ],
    () =>
      mdmAPI.getCertificates({
        team_id: currentTeamId,
        page: currentPage,
        per_page: 10,
      }),
    {
      ...DEFAULT_USE_QUERY_OPTIONS
      enabled: config?.mdm.android_enabled_and_configured ?? false,
    }
  );
  const profiles = profilesData?.profiles;
  const meta = profilesData?.meta;
  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }
    if (isLoadingProfiles) {
      return <Spinner />;
    }

    if (isErrorProfiles) {
      return <DataError />;
    }

    if (!profiles?.length) {
      return <AddProfileCard setShowModal={setShowAddProfileModal} />;
    }

    return (
      <>
        <UploadList
          keyAttribute="profile_uuid"
          listItems={profiles}
          HeadingComponent={() => (
            <ProfileListHeading
              onClickAddProfile={() => setShowAddProfileModal(true)}
            />
          )}
          ListItemComponent={({ listItem }) => (
            <ProfileListItem
              isPremium={!!isPremiumTier}
              profile={listItem}
              setProfileLabelsModalData={setProfileLabelsModalData}
              onClickInfo={onClickInfo}
              onClickDelete={onClickDelete}
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
    // premium message if Free
    // empty state if none
    // <UploadList />
  };

  return (
    <div className={`${baseClass}`}>
      <SectionHeader title="Certificates" alignLeftHeaderVertically />
      <PageDescription
        variant="right-panel"
        content={
          <>
            Deploy certificates. Currently only Android is supported. For macOS,
            iOS, iPadOS and Windows use configuration profiles, and for Linux
            use Scripts.
            <CustomLink
              newTab
              text="Learn more"
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/certificates`}
            />
          </>
        }
      />
      {renderContent()}
    </div>
  );
};

export default Certificates;
