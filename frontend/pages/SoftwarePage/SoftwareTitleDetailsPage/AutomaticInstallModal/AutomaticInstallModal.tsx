import React from "react";

import { ISoftware, ISoftwarePackagePolicy } from "interfaces/software";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";

const baseClass = "automatic-install-modal";

interface IPoliciesListItemProps {
  policy: ISoftwarePackagePolicy;
}

const PoliciesListItem = ({ policy }: IPoliciesListItemProps) => {
  return (
    <li key={policy.id} className={`${baseClass}__list-item`}>
      {policy.name}
    </li>
  );
};

interface IPoliciesListProps {
  policies: ISoftwarePackagePolicy[];
}

const PoliciesList = ({ policies }: IPoliciesListProps) => {
  return (
    <ul className={`${baseClass}__list`}>
      {policies.map((policy) => (
        <PoliciesListItem key={policy.id} policy={policy} />
      ))}
    </ul>
  );
};

interface IAutomaticInstallModalProps {
  policies: ISoftwarePackagePolicy[];
  onExit: () => void;
}

const AutomaticInstallModal = ({
  policies,
  onExit,
}: IAutomaticInstallModalProps) => {
  return (
    <Modal className={baseClass} title="Automatic install" onExit={onExit}>
      <>
        <p>
          Software will be installed when hosts fail this policy.{" "}
          <CustomLink
            newTab
            text="Learn more"
            url="https://fleetdm.com/learn-more-about/policy-automation-install-software"
          />
        </p>
        {policies.length > 0 && <PoliciesList policies={policies} />}
        <div className="modal-cta-wrap">
          <Button variant="brand">Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default AutomaticInstallModal;
