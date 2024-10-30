import React from "react";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { IMdmProfile, IProfileLabel } from "interfaces/mdm";
import InfoBanner from "components/InfoBanner";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";

interface IModalDescriptionProps {
  baseClass: string;
  profileName: string;
  targetType: "includeAll" | "excludeAny";
}

const ModalDescription = ({
  baseClass,
  profileName,
  targetType,
}: IModalDescriptionProps) => {
  const targetTypeText =
    targetType === "includeAll" ? (
      <>
        have <b>all</b>
      </>
    ) : (
      <>
        don&apos;t have <b>any</b>
      </>
    );

  return (
    <div className={`${baseClass}__description`}>
      <b>{profileName}</b> profile only applies to hosts that {targetTypeText}{" "}
      of these labels:
    </div>
  );
};

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
      targetTypeText = <b>have all</b>;
    } else {
      targetTypeText = <b>don&apos;t have any</b>;
    }

    return (
      <div className={`${baseClass}__description`}>
        <b>{name}</b> profile only applies to hosts that {targetTypeText} of
        these labels:
      </div>
    );
  };

  return (
    <Modal title="Custom target" onExit={() => setModalData(null)}>
      <div className={`${baseClass}__modal-content-wrap`}>
        {labels.some((label) => label.broken) && <BrokenLabelWarning />}
        <>{renderlabelDescription()}</>
        <LabelsList baseClass={baseClass} labels={labels} />
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
