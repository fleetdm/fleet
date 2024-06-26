import React from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { IMdmProfile, IProfileLabel } from "interfaces/mdm";
import InfoBanner from "components/InfoBanner";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";

const ModalDescription = ({
  baseClass,
  profileName,
}: {
  baseClass: string;
  profileName: string;
}) => (
  <div className={`${baseClass}__description`}>
    <b>{profileName}</b> will only be applied to hosts that have all these
    labels:
  </div>
);

const BrokenLabelWarning = () => (
  <InfoBanner color="yellow">
    <span>
      The configuration profile is{" "}
      <TooltipWrapper
        tipContent={`It won’t be applied to new hosts because one or more labels are deleted. To apply the profile to new hosts, please delete it and upload a new profile.`}
        underline
      >
        broken
      </TooltipWrapper>
      .
    </span>
  </InfoBanner>
);

const LabelsList = ({
  baseClass,
  labels,
}: {
  baseClass: string;
  labels: IProfileLabel[];
}) => (
  <div className={`${baseClass}__labels-list`}>
    {labels.map((label) => (
      <div key={label.name} className={`${baseClass}__labels-list--label`}>
        {label.name}
        {label.broken && (
          <span className={`${baseClass}__labels-list--label warning`}>
            <Icon name="warning" />
            Label deleted
          </span>
        )}
      </div>
    ))}
  </div>
);

interface IProfileLabelsModalProps {
  baseClass: string;
  profile: IMdmProfile | null;
  setModalData: React.Dispatch<React.SetStateAction<IMdmProfile | null>>;
}

const ProfileLabelsModal = ({
  baseClass,
  profile,
  setModalData,
}: IProfileLabelsModalProps) => {
  if (!profile?.labels?.length) {
    // caller ensures this never happens
    return null;
  }

  return (
    <Modal title="Custom target" onExit={() => setModalData(null)}>
      <div className={`${baseClass}__modal-content-wrap`}>
        {profile.labels.some((label) => label.broken) && <BrokenLabelWarning />}
        <ModalDescription baseClass={baseClass} profileName={profile.name} />
        <LabelsList baseClass={baseClass} labels={profile.labels} />
        <div className="modal-cta-wrap">
          <Button variant="brand" onClick={() => setModalData(null)}>
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ProfileLabelsModal;
