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
  <div className={`${baseClass}__labels-scroll`}>
    <div className={`${baseClass}__labels-list`}>
      {labels.map((label) => (
        <span key={label.name} className={`${baseClass}__label-badge`}>
          {label.name}
          {label.broken && (
            <span className={`${baseClass}__label-badge--broken`}>
              <Icon name="warning" size="small" />
              Label deleted
            </span>
          )}
        </span>
      ))}
    </div>
  </div>
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
    labels_include_all,
    labels_include_any,
    labels_exclude_any,
  } = profile;

  const includeLabels = labels_include_all || labels_include_any;
  const excludeLabels = labels_exclude_any;

  if (!includeLabels?.length && !excludeLabels?.length) {
    // caller ensures this never happens
    return null;
  }

  const allLabels = [...(includeLabels || []), ...(excludeLabels || [])];

  return (
    <Modal
      className={baseClass}
      title="Custom target"
      onExit={() => setModalData(null)}
    >
      <>
        {allLabels.some((label) => label.broken) && <BrokenLabelWarning />}
        <p className={`${baseClass}__description`}>
          <b>My includes/excludes </b>profile only applies to hosts that:
        </p>
        {!!includeLabels?.length && (
          <>
            <p className={`${baseClass}__section-title`}>
              <b>{labels_include_all ? "Include all" : "Include any"}</b> of
              these labels
            </p>
            <LabelsList labels={includeLabels} />
          </>
        )}
        {!!excludeLabels?.length && (
          <>
            <p className={`${baseClass}__section-title`}>
              <b>Exclude any</b> of these labels
            </p>
            <LabelsList labels={excludeLabels} />
          </>
        )}
        <div className="modal-cta-wrap">
          <Button onClick={() => setModalData(null)}>Close</Button>
        </div>
      </>
    </Modal>
  );
};

export default ProfileLabelsModal;
