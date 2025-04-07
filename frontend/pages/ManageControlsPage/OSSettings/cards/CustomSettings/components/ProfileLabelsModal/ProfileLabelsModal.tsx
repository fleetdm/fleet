import React from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { IMdmProfile, IProfileLabel } from "interfaces/mdm";
import InfoBanner from "components/InfoBanner";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";

const baseClass = "profile-labels-modal";

const BrokenLabelWarning = () => (
  <InfoBanner color="yellow">
    <span>
      The configuration profile is{" "}
      <TooltipWrapper
        tipContent={`It won't be applied to new hosts because one or more labels are deleted. To apply the profile to new hosts, please delete it and upload a new profile.`}
        underline
      >
        broken
      </TooltipWrapper>
      .
    </span>
  </InfoBanner>
);

const LabelsList = ({ labels }: { labels: IProfileLabel[] }) => (
  <ul className={`${baseClass}__labels-list`}>
    {labels.map((label) => (
      <li key={label.name} className={`${baseClass}__labels-list--label`}>
        {label.name}
        {label.broken && (
          <span className={`${baseClass}__labels-list--label warning`}>
            <Icon name="warning" />
            Label deleted
          </span>
        )}
      </li>
    ))}
  </ul>
);

interface IProfileLabelsModalProps {
  profile: IMdmProfile | null;
  setModalData: React.Dispatch<React.SetStateAction<IMdmProfile | null>>;
}

const ProfileLabelsModal = ({
  profile,
  setModalData,
}: IProfileLabelsModalProps) => {
  if (!profile) {
    return null;
  }

  const {
    name,
    labels_include_all,
    labels_include_any,
    labels_exclude_any,
  } = profile;
  const labels = labels_include_all || labels_include_any || labels_exclude_any;

  if (!labels?.length) {
    // caller ensures this never happens
    return null;
  }

  const renderlabelDescription = () => {
    let targetTypeText = <></>;
    if (labels_include_all) {
      targetTypeText = <b>have all</b>;
    } else if (labels_include_any) {
      targetTypeText = <b>have any</b>;
    } else {
      targetTypeText = <b>don&apos;t have any</b>;
    }

    return (
      <p className={`${baseClass}__description`}>
        <b>{name}</b> profile only applies to hosts that {targetTypeText} of
        these labels:
      </p>
    );
  };

  return (
    <Modal
      className={baseClass}
      title="Custom target"
      onExit={() => setModalData(null)}
    >
      <>
        {labels.some((label) => label.broken) && <BrokenLabelWarning />}
        <>{renderlabelDescription()}</>
        <LabelsList labels={labels} />
        <div className="modal-cta-wrap">
          <Button variant="brand" onClick={() => setModalData(null)}>
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default ProfileLabelsModal;
