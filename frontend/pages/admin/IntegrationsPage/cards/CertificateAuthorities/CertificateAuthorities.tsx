import React, { useContext, useMemo, useState } from "react";
import { Link } from "react-router";

import paths from "router/paths";
import { AppContext } from "context/app";
import { ICertificateIntegration } from "interfaces/integration";
import createMockConfig from "__mocks__/configMock";

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

const baseClass = "certificate-authorities";

const CertificateAuthorities = () => {
  let { config } = useContext(AppContext);
  config = createMockConfig({
    integrations: {
      zendesk: [],
      jira: [],
      digicert: [
        {
          name: "DigiCert CA",
          id: 1,
          api_token: "123456",
          profile_id: "7ed77396-9186-4bfa-9fa7-63dddc46b8a3",
          certificate_common_name:
            "$FLEET_VAR_HOST_HARDWARE_SERIAL@example.com",
          certificate_user_principal_names: ["$FLEET_VAR_HOST_HARDWARE_SERIAL"],
          certificate_seat_id: "$FLEET_VAR_HOST_HARDWARE_SERIAL@example.com",
        },
      ],
      ndes_scep_proxy: {
        url: "https://ndes.scep.com",
        admin_url: "https://ndes.scep.com/admin",
        username: "ndes",
        password: "password",
      },
      custom_scep_proxy: [
        {
          id: 1,
          name: "Custom SCEP Proxy",
          server_url: "https://custom.scep.com",
          challenge: "challenge",
        },
        {
          id: 2,
          name: "Custom SCEP Proxy 2",
          server_url: "https://custom.scep2.com",
          challenge: "challenge-2",
        },
      ],
    },
  });

  const [showAddCertAuthorityModal, setShowAddCertAuthorityModal] = useState(
    false
  );
  const [showEditCertAuthorityModal, setShowEditCertAuthorityModal] = useState(
    false
  );
  const [
    showDeleteCertAuthoirtyModal,
    setShowDeleteCertAuthorityModal,
  ] = useState(false);

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
    // TODO: use useCallback
    const certAuthority = getCertificateAuthority(
      cert.id,
      config?.integrations.ndes_scep_proxy,
      config?.integrations.digicert,
      config?.integrations.custom_scep_proxy
    );
    setSelectedCertAuthority(certAuthority);
    setShowEditCertAuthorityModal(true);
  };

  const onDeleteCertAuthority = (cert: ICertAuthorityListData) => {
    // TODO: use useCallback
    const certAuthority = getCertificateAuthority(
      cert.id,
      config?.integrations.ndes_scep_proxy,
      config?.integrations.digicert,
      config?.integrations.custom_scep_proxy
    );
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
      {showAddCertAuthorityModal && <div>Modal showing</div>}
      {showEditCertAuthorityModal && <div>Modal showing</div>}
      {showDeleteCertAuthoirtyModal && selectedCertAuthority && (
        <DeleteCertificateAuthorityModal
          certAuthority={selectedCertAuthority}
          onExit={() => setShowDeleteCertAuthorityModal(false)}
        />
      )}
    </div>
  );
};

export default CertificateAuthorities;
