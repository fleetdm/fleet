import React, { useState, useCallback, useContext } from "react";
import { AxiosError } from "axios";
import { useQuery } from "react-query";
import { formatDistanceToNow } from "date-fns";

import { AppContext } from "context/app";
import PATHS from "router/paths";

import { createMockAndroidCert } from "__mocks__/certificatesMock";

import UploadList from "pages/ManageControlsPage/components/UploadList";
import UploadListHeading from "pages/ManageControlsPage/components/UploadListHeading";

import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import ListItem from "components/ListItem";
import Pagination from "components/Pagination";
import CustomLink from "components/CustomLink";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import PageDescription from "components/PageDescription";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SectionHeader from "components/SectionHeader";
import GenericMsgWithNavButton from "components/GenericMsgWithNavButton";

import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

import certAPI, {
  ICertificate,
  IGetCertsResponse,
  IQueryKeyGetCerts,
} from "services/entities/certificates";

import { IOSSettingsCommonProps } from "../../OSSettingsNavItems";
import AddCertCard from "./components/AddCertificateCard/AddCertificateCard";
import DeleteCertModal from "./components/DeleteCertificateModal";
import AddCertModal from "./components/AddCertificateModal";

const baseClass = "certificates";

export type ICertificatesProps = IOSSettingsCommonProps & {
  currentPage?: number;
};

const Certificates = ({
  currentTeamId,
  router,
  currentPage = 0,
  onMutation,
}: ICertificatesProps) => {
  const [showAddCertModal, setShowAddCertModal] = useState(false);
  const [certToDelete, setCertToDelete] = useState<null | ICertificate>(null);
  const { config, isPremiumTier } = useContext(AppContext);

  // const androidMdmEnabled = !!config?.mdm.android_enabled_and_configured;
  const androidMdmEnabled = true;

  let {
    data: certsResp,
    isLoading: isLoadingCerts,
    isError: isErrorCerts,
    refetch: refetchCerts,
  } = useQuery<
    IGetCertsResponse,
    AxiosError,
    IGetCertsResponse,
    IQueryKeyGetCerts[]
  >(
    [
      {
        scope: "certificates",
        team_id: currentTeamId,
        page: currentPage,
        per_page: 10,
      },
    ],
    ({ queryKey }) => certAPI.getCerts(queryKey[0]),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier && androidMdmEnabled,
    }
  );

  // TODO - undo
  certsResp = {
    certificates: [createMockAndroidCert()],
    meta: {
      has_next_results: false,
      has_previous_results: false,
    },
  };

  const certs = certsResp?.certificates;
  const { has_next_results: hasNext, has_previous_results: hasPrev } =
    certsResp?.meta || {};

  const onAddSuccess = () => {
    refetchCerts();
    onMutation();
  };

  // const onDeleteCert = async (profileId: string) => {
  //   setIsDeleting(true);
  //   try {
  //     await mdmAPI.deleteProfile(profileId);
  //     refetchCerts();
  //     onMutation();
  //     renderFlash("success", "Successfully deleted!");
  //   } catch (e) {
  //     renderFlash("error", "Couldn't delete. Please try again.");
  //   } finally {
  //     selectedProfile.current = null;
  //     setShowDeleteProfileModal(false);
  //   }
  //   setIsDeleting(false);
  // };

  // pagination controls
  const path = PATHS.CONTROLS_CUSTOM_SETTINGS;
  const queryString = isPremiumTier ? `?team_id=${currentTeamId}&` : "?";

  const onPrevPage = useCallback(() => {
    router.push(path.concat(`${queryString}page=${currentPage - 1}`));
  }, [router, path, currentPage, queryString]);

  const onNextPage = useCallback(() => {
    router.push(path.concat(`${queryString}page=${currentPage + 1}`));
  }, [router, path, currentPage, queryString]);

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }
    if (!androidMdmEnabled) {
      return (
        <GenericMsgWithNavButton
          header="Manage your hosts"
          buttonText="Turn on"
          path={PATHS.ADMIN_INTEGRATIONS_MDM}
          router={router}
          info="Android MDM must be turned on to apply custom settings."
        />
      );
    }
    if (isLoadingCerts) {
      return <Spinner />;
    }

    if (isErrorCerts) {
      return <DataError />;
    }

    if (!certs?.length) {
      return <AddCertCard setShowModal={setShowAddCertModal} />;
    }

    return (
      <>
        <UploadList
          keyAttribute="id"
          listItems={certs || []}
          HeadingComponent={() => (
            <UploadListHeading
              entityName="Certificate"
              createEntityText="Create"
              onClickAdd={() => setShowAddCertModal(true)}
            />
          )}
          ListItemComponent={({ listItem }) => {
            const {
              name,
              certificate_authority_name: caName,
              created_at,
            } = listItem;

            const details = (
              <>
                {caName} &bull; Uploaded{" "}
                {formatDistanceToNow(new Date(created_at))} ago
              </>
            );
            return (
              <ListItem
                graphic="file-certificate"
                title={name}
                details={details}
                actions={
                  <GitOpsModeTooltipWrapper
                    renderChildren={(disableChildren) => (
                      <Button
                        disabled={disableChildren}
                        className={`${baseClass}__delete-button`}
                        variant="icon"
                        onClick={() => setCertToDelete(listItem)}
                      >
                        <Icon name="trash" />
                      </Button>
                    )}
                  />
                }
              />
            );
          }}
        />
        <Pagination
          disableNext={!hasNext}
          disablePrev={!hasPrev}
          hidePagination={!hasNext && !hasPrev}
          onNextPage={onNextPage}
          onPrevPage={onPrevPage}
        />
      </>
    );
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
      {showAddCertModal && (
        <AddCertModal
          existingCerts={certs}
          onExit={() => setShowAddCertModal(false)}
          onSuccess={onAddSuccess}
          currentTeamId={currentTeamId}
        />
      )}
      {certToDelete && (
        <DeleteCertModal
          cert={certToDelete}
          onExit={() => setCertToDelete(null)}
        />
      )}
    </div>
  );
};

export default Certificates;
