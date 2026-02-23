import React, { useMemo } from "react";

import { ICertificateAuthorityPartial } from "interfaces/certificates";

import UploadList from "pages/ManageControlsPage/components/UploadList";

import CertAuthorityListHeader from "../CertAuthorityListHeader";
import CertAuthorityListItem from "../CertAuthorityListItem";
import CA_LABEL_BY_TYPE from "../helpers";

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
    return {
      ...cert,
      description: CA_LABEL_BY_TYPE[cert.type],
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
