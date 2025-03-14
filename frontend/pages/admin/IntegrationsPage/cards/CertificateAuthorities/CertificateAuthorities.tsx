import React, { useContext, useMemo, useState } from "react";
import { Link } from "react-router";

import paths from "router/paths";
import { AppContext } from "context/app";
import { ICertificateIntegration } from "interfaces/integration";

import SectionHeader from "components/SectionHeader";
import CustomLink from "components/CustomLink";

import CertificateAuthorityList from "./components/CertificateAuthorityList";
import {
  generateListData,
  getCertificateAuthority,
  ICertAuthorityListData,
} from "./helpers";
import AddCertAuthorityCard from "./components/AddCertAuthorityCard";
import DeleteCertificateAuthorityModal from "./components/DeleteCertificateAuthorityModal";
import AddCertAuthorityModal from "./components/AddCertAuthorityModal";
import EditCertAuthorityModal from "./components/EditCertAuthorityModal";

const baseClass = "certificate-authorities";

const CertificateAuthorities = () => {
  const { config } = useContext(AppContext);

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

  const [selectedListItemId, setSelectedListItemId] = useState<string | null>(
    null
  );
  const [
    selectedCertAuthority,
    setSelectedCertAuthority,
  ] = useState<ICertificateIntegration | null>(null);

  const certificateAuthorities = useMemo(() => {
    if (!config) return [];
    return generateListData(
      config?.integrations.ndes_scep_proxy,
      config?.integrations.digicert,
      config?.integrations.custom_scep_proxy
    );
  }, [config]);

  const onAddCertAuthority = () => {
    setShowAddCertAuthorityModal(true);
  };

  const onEditCertAuthority = (cert: ICertAuthorityListData) => {
    const certAuthority = getCertificateAuthority(
      cert.id,
      config?.integrations.ndes_scep_proxy,
      config?.integrations.digicert,
      config?.integrations.custom_scep_proxy
    );
    setSelectedListItemId(cert.id);
    setSelectedCertAuthority(certAuthority);
    setShowEditCertAuthorityModal(true);
  };

  const onDeleteCertAuthority = (cert: ICertAuthorityListData) => {
    const certAuthority = getCertificateAuthority(
      cert.id,
      config?.integrations.ndes_scep_proxy,
      config?.integrations.digicert,
      config?.integrations.custom_scep_proxy
    );
    setSelectedListItemId(cert.id);
    setSelectedCertAuthority(certAuthority);
    setShowDeleteCertAuthorityModal(true);
  };

  const renderContent = () => {
    if (certificateAuthorities.length === 0) {
      return <AddCertAuthorityCard onAddCertAuthority={onAddCertAuthority} />;
    }

    return (
      <CertificateAuthorityList
        certAuthorities={certificateAuthorities}
        onAddCertAuthority={onAddCertAuthority}
        onClickEdit={onEditCertAuthority}
        onClickDelete={onDeleteCertAuthority}
      />
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Certificates" />
      <p className={`${baseClass}__page-description`}>
        To help your end users connect to Wi-Fi or VPNs, you can add your
        certificate authority. Then, head over to{" "}
        <Link to={paths.CONTROLS_CUSTOM_SETTINGS}>
          Controls {">"} OS Settings {">"} Custom
        </Link>{" "}
        settings to configure how certificates are delivered to your hosts.{" "}
        <CustomLink
          text="Learn more"
          url="https://fleetdm.com/learn-more-about/certificate-authorities"
          newTab
        />
      </p>
      {renderContent()}
      {showAddCertAuthorityModal && (
        <AddCertAuthorityModal
          onExit={() => setShowAddCertAuthorityModal(false)}
        />
      )}
      {showEditCertAuthorityModal && selectedCertAuthority && (
        <EditCertAuthorityModal
          certAuthority={selectedCertAuthority}
          onExit={() => setShowEditCertAuthorityModal(false)}
        />
      )}
      {showDeleteCertAuthorityModal &&
        selectedCertAuthority &&
        selectedListItemId && (
          <DeleteCertificateAuthorityModal
            listItemId={selectedListItemId}
            certAuthority={selectedCertAuthority}
            onExit={() => setShowDeleteCertAuthorityModal(false)}
          />
        )}
    </div>
  );
};

export default CertificateAuthorities;
