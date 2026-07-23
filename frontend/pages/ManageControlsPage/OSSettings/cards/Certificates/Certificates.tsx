import React, { useState, useCallback, useContext } from "react";
import { AxiosError } from "axios";
import { useQuery } from "react-query";
import { timeAgo } from "utilities/date_format";

import { AppContext } from "context/app";
import PATHS from "router/paths";
import useGitOpsMode from "hooks/useGitOpsMode";
import { getGitOpsModeTipContent } from "utilities/helpers";

import { IDropdownOption } from "interfaces/dropdownOption";

import UploadList from "components/UploadList";
import UploadListHeading from "pages/ManageControlsPage/components/UploadListHeading";

import ActionsDropdown from "components/ActionsDropdown";
import Button from "components/buttons/Button";
import ListItem from "components/ListItem";
import Pagination from "components/Pagination";
import CustomLink from "components/CustomLink";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import PageDescription from "components/PageDescription";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SectionHeader from "components/SectionHeader";
import EmptyState from "components/EmptyState";
import TooltipTruncatedText from "components/TooltipTruncatedText";

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
import AddCertAuthorityCard from "./components/AddCertAuthorityCard";
import DeleteCertModal from "./components/DeleteCertificateModal";
import AddCertModal from "./components/AddCertificateModal";
import ViewCertModal from "./components/ViewCertificateModal";

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
  const [certToView, setCertToView] = useState<null | ICertificate>(null);
  const [certToDelete, setCertToDelete] = useState<null | ICertificate>(null);
  const { config, isPremiumTier } = useContext(AppContext);
  const { gitOpsModeEnabled, repoURL } = useGitOpsMode();

  const androidMdmEnabled = !!config?.mdm.android_enabled_and_configured;

  const {
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
        fleet_id: currentTeamId,
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

  const {
    data: certAuthorities,
    isLoading: isLoadingCAs,
    isError: isErrorCAs,
  } = useQuery(
    ["certAuthorities"],
    () => certAPI.getCertificateAuthoritiesList(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier && androidMdmEnabled,
      select: (data) => data.certificate_authorities,
    }
  );

  const hasCustomScepCA = (certAuthorities ?? []).some(
    (ca) => ca.type === "custom_scep_proxy"
  );

  const certs = certsResp?.certificates || [];
  const { has_next_results: hasNext, has_previous_results: hasPrev } =
    certsResp?.meta || {};

  const onUpdateSuccess = () => {
    refetchCerts();
    onMutation();
  };

  // pagination controls
  const path = PATHS.CONTROLS_CERTIFICATES;
  const queryString = isPremiumTier ? `?fleet_id=${currentTeamId}&` : "?";

  const onPrevPage = useCallback(() => {
    router.push(path.concat(`${queryString}page=${currentPage - 1}`));
  }, [router, path, currentPage, queryString]);

  const onNextPage = useCallback(() => {
    router.push(path.concat(`${queryString}page=${currentPage + 1}`));
  }, [router, path, currentPage, queryString]);

  const onSelectCertAction = (action: string, cert: ICertificate) => {
    switch (action) {
      case "view":
        setCertToView(cert);
        break;
      case "delete":
        setCertToDelete(cert);
        break;
      default:
        break;
    }
  };

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }
    if (!androidMdmEnabled) {
      return (
        <EmptyState
          variant="header-list"
          header="Additional configuration required"
          info="Android MDM must be turned on to add certificates."
          primaryButton={
            <Button onClick={() => router.push(PATHS.ADMIN_INTEGRATIONS_MDM)}>
              Turn on
            </Button>
          }
        />
      );
    }
    if (isLoadingCerts) {
      return <Spinner />;
    }

    if (isErrorCerts) {
      return <DataError />;
    }

    if (!certs.length) {
      if (isLoadingCAs) {
        return <Spinner />;
      }
      if (isErrorCAs) {
        return <DataError />;
      }
      return hasCustomScepCA ? (
        <AddCertCard setShowModal={setShowAddCertModal} />
      ) : (
        <AddCertAuthorityCard router={router} />
      );
    }

    return (
      <>
        <UploadList
          keyAttribute="id"
          listItems={certs}
          HeadingComponent={() => (
            <UploadListHeading
              entityName="Certificate"
              createEntityText="Add"
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
                {caName} &bull; Updated{" "}
                {timeAgo(new Date(created_at), { addSuffix: true })}
              </>
            );

            const certActions: IDropdownOption[] = [
              { label: "View certificate", value: "view" },
              {
                label: "Delete",
                value: "delete",
                disabled: gitOpsModeEnabled,
                tooltipContent:
                  gitOpsModeEnabled && repoURL
                    ? getGitOpsModeTipContent(repoURL)
                    : undefined,
              },
            ];

            return (
              <ListItem
                graphic="file-certificate"
                title={<TooltipTruncatedText value={name} />}
                details={details}
                actions={
                  <ActionsDropdown
                    options={certActions}
                    placeholder="Actions"
                    variant="secondary"
                    menuAlign="right"
                    menuPlacement="auto"
                    onChange={(action) => onSelectCertAction(action, listItem)}
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
            use scripts.{" "}
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
          onExit={() => setShowAddCertModal(false)}
          onSuccess={onUpdateSuccess}
          currentTeamId={currentTeamId}
        />
      )}
      {certToView && (
        <ViewCertModal cert={certToView} onExit={() => setCertToView(null)} />
      )}
      {certToDelete && (
        <DeleteCertModal
          cert={certToDelete}
          onSuccess={onUpdateSuccess}
          onExit={() => setCertToDelete(null)}
        />
      )}
    </div>
  );
};

export default Certificates;
