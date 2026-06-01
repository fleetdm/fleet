import React, { useContext, useState } from "react";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import { ICertificateAuthorityPartial } from "interfaces/certificates";
import certificatesAPI from "services/entities/certificates";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import SettingsSection from "pages/admin/components/SettingsSection";
import PageDescription from "components/PageDescription";
import CustomLink from "components/CustomLink";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

import Button from "components/buttons/Button";
import EmptyState from "components/EmptyState";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import CertificateAuthorityList from "./components/CertificateAuthorityList";
import DeleteCertificateAuthorityModal from "./components/DeleteCertificateAuthorityModal";
import AddCertAuthorityModal from "./components/AddCertAuthorityModal";
import EditCertAuthorityModal from "./components/EditCertAuthorityModal";

const CertificateAuthorities = () => {
  const { isPremiumTier } = useContext(AppContext);

  const [showAddCertAuthorityModal, setShowAddCertAuthorityModal] = useState(
    false
  );
  const [showEditCertAuthorityModal, setShowEditCertAuthorityModal] = useState(
    false
  );
  const [
    showDeleteCertAuthorityModal,
    setShowDeleteCertAuthorityModal,
  ] = useState(false);

  const [
    selectedCertAuthority,
    setSelectedCertAuthority,
  ] = useState<ICertificateAuthorityPartial | null>(null);

  const {
    data: certAuthorities,
    isLoading,
    isError,
    refetch: refetchCertAuthorities,
  } = useQuery(
    "certAuthorities",
    () => {
      return certificatesAPI.getCertificateAuthoritiesList();
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (data) => data.certificate_authorities,
    }
  );

  const onAddCertAuthority = () => {
    setShowAddCertAuthorityModal(true);
  };

  const onAddedNewCertAuthority = () => {
    refetchCertAuthorities();
    setShowAddCertAuthorityModal(false);
  };

  const onEditCertAuthority = (cert: ICertificateAuthorityPartial) => {
    setSelectedCertAuthority(cert);
    setShowEditCertAuthorityModal(true);
  };

  const onEditedCertAuthority = () => {
    refetchCertAuthorities();
    setShowEditCertAuthorityModal(false);
  };

  const onDeleteCertAuthority = (cert: ICertificateAuthorityPartial) => {
    setSelectedCertAuthority(cert);
    setShowDeleteCertAuthorityModal(true);
  };

  const onDeletedCertAuthority = () => {
    refetchCertAuthorities();
    setShowDeleteCertAuthorityModal(false);
  };

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }

    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError />;
    }

    const pageDescription = (
      <PageDescription
        variant="right-panel"
        content={
          <>
            To help your end users connect to Wi-Fi or VPNs, you can add your
            certificate authority.{" "}
            <CustomLink
              text="Learn more"
              url="https://fleetdm.com/learn-more-about/certificate-authorities"
              newTab
            />
          </>
        }
      />
    );

    if (certAuthorities === undefined || certAuthorities.length === 0) {
      return (
        <>
          {pageDescription}
          <EmptyState
            variant="header-list"
            header="No certificate authorities"
            info="Add a certificate authority (CA) to help end users connect to Wi-Fi or VPNs."
            primaryButton={
              <GitOpsModeTooltipWrapper
                renderChildren={(disableChildren) => (
                  <Button
                    disabled={disableChildren}
                    onClick={onAddCertAuthority}
                  >
                    Add certificate authority
                  </Button>
                )}
              />
            }
          />
        </>
      );
    }

    return (
      <>
        {pageDescription}
        <CertificateAuthorityList
          certAuthorities={certAuthorities}
          onAddCertAuthority={onAddCertAuthority}
          onClickEdit={onEditCertAuthority}
          onClickDelete={onDeleteCertAuthority}
        />
      </>
    );
  };

  return (
    <SettingsSection title="Certificate authorities">
      {renderContent()}
      {showAddCertAuthorityModal && certAuthorities && (
        <AddCertAuthorityModal
          certAuthorities={certAuthorities}
          onExit={onAddedNewCertAuthority}
        />
      )}
      {showEditCertAuthorityModal &&
        selectedCertAuthority &&
        certAuthorities && (
          <EditCertAuthorityModal
            certAuthority={selectedCertAuthority}
            onExit={onEditedCertAuthority}
          />
        )}
      {showDeleteCertAuthorityModal && selectedCertAuthority && (
        <DeleteCertificateAuthorityModal
          certAuthority={selectedCertAuthority}
          onExit={onDeletedCertAuthority}
        />
      )}
    </SettingsSection>
  );
};

export default CertificateAuthorities;
