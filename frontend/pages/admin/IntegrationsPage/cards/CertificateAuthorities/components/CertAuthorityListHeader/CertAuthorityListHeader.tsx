import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon";
import React from "react";

const baseClass = "cert-authority-list-header";

interface ICertAuthorityListHeaderProps {
  onClickAddCertAuthority: () => void;
}

const CertAuthorityListHeader = ({
  onClickAddCertAuthority,
}: ICertAuthorityListHeaderProps) => {
  return (
    <div className={baseClass}>
      <span className={`${baseClass}__name`}>Certificate authority (CA)</span>
      <span className={`${baseClass}__actions`}>
        <GitOpsModeTooltipWrapper
          position="left"
          renderChildren={(disableChildren) => (
            <Button
              disabled={disableChildren}
              variant="text-icon"
              className={`${baseClass}__add-button`}
              onClick={onClickAddCertAuthority}
            >
              <span className={`${baseClass}__btn-label`}>
                <Icon name="plus" />
                <span>Add CA</span>
              </span>
            </Button>
          )}
        />
      </span>
    </div>
  );
};

export default CertAuthorityListHeader;
