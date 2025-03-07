import React from "react";

import UploadList from "pages/ManageControlsPage/components/UploadList";

import CertAuthorityListHeader from "../CertAuthorityListHeader";
import CertAuthorityListItem from "../CertAuthorityListItem";
import { ICertAuthority } from "../../helpers";

const baseClass = "certificate-authority-list";

interface ICertificateAuthorityListProps {
  certAuthorities: ICertAuthority[];
  onAddCertAuthority: () => void;
  onClickEdit: (cert: ICertAuthority) => void;
  onClickDelete: (cert: ICertAuthority) => void;
}

const CertificateAuthorityList = ({
  certAuthorities,
  onAddCertAuthority,
  onClickEdit,
  onClickDelete,
}: ICertificateAuthorityListProps) => {
  return (
    <UploadList<ICertAuthority>
      className={baseClass}
      keyAttribute="name"
      listItems={certAuthorities}
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
