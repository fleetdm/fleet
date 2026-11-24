import { AxiosError } from "axios";
import React, { useState, useCallback, useContext } from "react";
import { useQuery } from "react-query";

import ActionsDropdown from "components/ActionsDropdown";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import ListItem from "components/ListItem";
import PageDescription from "components/PageDescription";
import Pagination from "components/Pagination";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import UploadList from "components/UploadList";
import { AppContext } from "context/app";
import useGitOpsMode from "hooks/useGitOpsMode";
import { IDropdownOption } from "interfaces/dropdownOption";
import PATHS from "router/paths";
import certAPI, {
  IAndroidCACertificate,
  ICertificate,
  IGetCertsResponse,
  IQueryKeyGetCerts,
} from "services/entities/certificates";
import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";
import { timeAgo } from "utilities/date_format";
import { getGitOpsModeTipContent } from "utilities/helpers";

import { IOSSettingsCommonProps } from "../../OSSettingsNavItems";

import AddCertAuthorityCard from "./components/AddCertAuthorityCard";
import AddCertCard from "./components/AddCertificateCard/AddCertificateCard";
import AddCertModal from "./components/AddCertificateModal";
import DeleteCertModal from "./components/DeleteCertificateModal";
import ViewCertModal from "./components/ViewCertificateModal";

const baseClass = "certificates";

export type ICertificatesProps = IOSSettingsCommonProps & {
  currentPage?: number;
};

type CertItem =
  | (ICertificate & { isCA?: false })
  | (IAndroidCACertificate & {
      isCA: true;
      certificate_authority_name?: string;
    });

const Certificates = ({
  currentTeamId,
  router,
  currentPage = 0,
  onMutation,
}: ICertificatesProps) => {
  const [showAddCertModal, setShowAddCertModal] = useState(false);
  const [certToView, setCertToView] = useState<null | ICertificate>(null);
  const [certToDelete, setCertToDelete] = useState<null | {
    id: number;
    name: string;
    isCA?: boolean;
  }>(null);
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
    data: caCertsResp,
    isLoading: isLoadingCACerts,
    isError: isErrorCACerts,
    refetch: refetchCACerts,
  } = useQuery(
    ["caCertificates", currentTeamId],
    () => certAPI.getCACerts(currentTeamId),
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

  const scepCerts: CertItem[] = (certsResp?.certificates || []).map((c) => ({
    ...c,
    isCA: false as const,
  }));
  const caCerts: CertItem[] = (caCertsResp?.ca_certificates || []).map(
    (ca) => ({
      ...ca,
      isCA: true as const,
    })
  );
  const allCerts = [...scepCerts, ...caCerts];

  const { has_next_results: hasNext, has_previous_results: hasPrev } =
    certsResp?.meta || {};

  const onUpdateSuccess = () => {
    refetchCerts();
    refetchCACerts();
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

  const onSelectCertAction = (action: string, cert: CertItem) => {
    switch (action) {
      case "view":
        if (!cert.isCA) {
          setCertToView(cert as ICertificate);
        }
        break;
      case "delete":
        setCertToDelete({ id: cert.id, name: cert.name, isCA: cert.isCA });
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
    if (isLoadingCerts || isLoadingCACerts) {
      return <Spinner />;
    }

    if (isErrorCerts || isErrorCACerts) {
      return <DataError />;
    }

    if (!allCerts.length) {
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
          listItems={allCerts}
          ListItemComponent={({ listItem }) => {
            const isCA = "isCA" in listItem && listItem.isCA;
            let details: React.ReactNode;
            if (isCA) {
              const caItem = listItem as IAndroidCACertificate;
              const expiry = caItem.not_valid_after
                ? ` • Expires ${new Date(
                    caItem.not_valid_after
                  ).toLocaleDateString("en-US", {
                    month: "short",
                    day: "numeric",
                    year: "numeric",
                  })}`
                : "";
              details = (
                <>
                  Root/intermediate CA &bull; Updated{" "}
                  {timeAgo(new Date(caItem.updated_at || caItem.created_at), {
                    addSuffix: true,
                  })}
                  {expiry}
                </>
              );
            } else {
              details = (
                <>
                  {listItem.certificate_authority_name} &bull; Updated{" "}
                  {timeAgo(new Date(listItem.created_at), { addSuffix: true })}
                </>
              );
            }

            const certActions: IDropdownOption[] = [
              ...(!isCA ? [{ label: "View certificate", value: "view" }] : []),
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
                title={<TooltipTruncatedText value={listItem.name} />}
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

  const showAddCertButton =
    isPremiumTier &&
    androidMdmEnabled &&
    (hasCustomScepCA || allCerts.length > 0);

  return (
    <div className={`${baseClass}`}>
      <SectionHeader title="Certificates" alignLeftHeaderVertically />
      <div className={`${baseClass}__tab-header`}>
        <PageDescription
          variant="right-panel"
          content={
            <>
              Deploy certificates. Currently only Android is supported. For
              macOS, iOS, iPadOS and Windows use configuration profiles, and for
              Linux use scripts.{" "}
              <CustomLink
                newTab
                text="Learn more"
                url={`${LEARN_MORE_ABOUT_BASE_LINK}/certificates`}
              />
            </>
          }
        />
        {showAddCertButton && (
          <GitOpsModeTooltipWrapper
            position="left"
            renderChildren={(disableChildren) => (
              <Button
                variant="secondary"
                size="small"
                onClick={() => setShowAddCertModal(true)}
                disabled={disableChildren}
                icon="plus"
              >
                Add certificate
              </Button>
            )}
          />
        )}
      </div>
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
