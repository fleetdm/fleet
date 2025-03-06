import React from "react";
import { Link } from "react-router";

import paths from "router/paths";

import SectionHeader from "components/SectionHeader";
import CustomLink from "components/CustomLink";
import Card from "components/Card";
import Button from "components/buttons/Button";

import CertificateList from "./components/CertificateList";

const baseClass = "certificates-integration";

interface IAddCertCardProps {
  onAddCert: () => void;
}

const AddCertCard = ({ onAddCert }: IAddCertCardProps) => {
  return (
    <Card paddingSize="xxlarge" className={`${baseClass}__add-cert-card`}>
      <div className={`${baseClass}__add-cert-card-content`}>
        <p className={`${baseClass}__add-cert-card-title`}>
          Add your certificate authority (CA)
        </p>
        <p>Help your end users connect to Wi-Fi or VPNs.</p>
      </div>
      <Button
        className={`${baseClass}__add-cert-card-button`}
        onClick={onAddCert}
      >
        Add CA
      </Button>
    </Card>
  );
};

interface ICertificatesProps {}

const certs = [];

const Certificates = ({}: ICertificatesProps) => {
  const onAddCert = () => {
    console.log("Add cert");
  };

  const renderContent = () => {
    if (certs.length === 0) {
      return <AddCertCard onAddCert={onAddCert} />;
    }

    return <CertificateList />;
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
    </div>
  );
};

export default Certificates;
