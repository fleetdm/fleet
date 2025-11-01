import React, { useMemo } from "react";

import { ICertificateAuthorityPartial } from "interfaces/certificates";

import UploadList from "pages/ManageControlsPage/components/UploadList";

import CertAuthorityListHeader from "../CertAuthorityListHeader";
import CertAuthorityListItem from "../CertAuthorityListItem";

const baseClass = "certificate-authority-list";

export type ICertAuthorityListData = ICertificateAuthorityPartial & {
  description: string;
};

/** This function extends the ICertificateAuthorityPartial with a description field
 * to provide a user-friendly description for each certificate authority.
 */
export const generateListData = (
  certAuthorities: ICertificateAuthorityPartial[]
) => {
  return certAuthorities.map<ICertAuthorityListData>((cert) => {
    let description = "";
    switch (cert.type) {
      case "ndes_scep_proxy":
        description = "Microsoft Network Device Enrollment Service (NDES)";
        break;
      case "digicert":
        description = "DigiCert";
        break;
      case "custom_scep_proxy":
        description = "Custom Simple Certificate Enrollment Protocol (SCEP)";
        break;
      case "hydrant":
        description = "Hydrant (EST - Enrollment Over Secure Transport) ";
        break;
      case "smallstep":
        description = "Smallstep";
        break;
      case "custom_est":
        description = "Custom Enrollment Over Secure Transport (EST)";
        break;
      default:
        description = "Unknown Certificate Authority Type";
    }

    return {
      ...cert,
      description,
    };
  });
};

interface ICertificateAuthorityListProps {
  certAuthorities: ICertificateAuthorityPartial[];
  onAddCertAuthority: () => void;
  onClickEdit: (cert: ICertificateAuthorityPartial) => void;
  onClickDelete: (cert: ICertificateAuthorityPartial) => void;
}

const CertificateAuthorityList = ({
  certAuthorities,
  onAddCertAuthority,
  onClickEdit,
  onClickDelete,
}: ICertificateAuthorityListProps) => {
  const listData = useMemo(() => generateListData(certAuthorities), [
    certAuthorities,
  ]);

  return (
    <UploadList<ICertAuthorityListData>
      className={baseClass}
      keyAttribute="name"
      listItems={listData}
      HeadingComponent={() => (
        <CertAuthorityListHeader onClickAddCertAuthority={onAddCertAuthority} />
      )}
      ListItemComponent={({ listItem }) => (
        <CertAuthorityListItem
          cert={listItem}
          onClickEdit={() => onClickEdit(listItem)}
          onClickDelete={() => onClickDelete(listItem)}
        />
      )}
    />
  );
};

export default CertificateAuthorityList;
