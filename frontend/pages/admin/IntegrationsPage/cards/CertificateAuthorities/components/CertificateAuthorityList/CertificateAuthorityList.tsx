import React from "react";

import UploadList from "pages/ManageControlsPage/components/UploadList";

import CertAuthorityListHeader from "../CertAuthorityListHeader";
import CertAuthorityListItem from "../CertAuthorityListItem";
import { ICertAuthorityListData } from "../../helpers";

const baseClass = "certificate-authority-list";

interface ICertificateAuthorityListProps {
  certAuthorities: ICertAuthorityListData[];
  onAddCertAuthority: () => void;
  onClickEdit: (cert: ICertAuthorityListData) => void;
  onClickDelete: (cert: ICertAuthorityListData) => void;
}

const CertificateAuthorityList = ({
  certAuthorities,
  onAddCertAuthority,
  onClickEdit,
  onClickDelete,
}: ICertificateAuthorityListProps) => {
  return (
    <UploadList<ICertAuthorityListData>
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
