import React from "react";

// @ts-ignore
import EditPackForm from "components/forms/packs/EditPackForm";
import { IPack } from "interfaces/pack";
import { IQuery } from "interfaces/query";
import { ITarget } from "interfaces/target";

const baseClass = "edit-pack-form";

interface IEditPackFormWrapper {
  className?: string;
  handleSubmit?: (formData: any) => void;
  onCancelEditPack: () => void;
  onEditPack: () => void;
  onFetchTargets?: (query: IQuery, targetsResponse: any) => boolean;
  pack: IPack;
  packTargets?: ITarget[];
  targetsCount?: number;
  isPremiumTier?: boolean;
}

const EditPackFormWrapper = (props: IEditPackFormWrapper): JSX.Element => {
  const {
    className,
    handleSubmit,
    onCancelEditPack,
    onFetchTargets,
    pack,
    packTargets,
    targetsCount,
    isPremiumTier,
  } = props;

  return (
    <EditPackForm
      className={className}
      formData={{ ...pack, targets: packTargets }}
      handleSubmit={handleSubmit}
      onCancel={onCancelEditPack}
      onFetchTargets={onFetchTargets}
      targetsCount={targetsCount}
      isPremiumTier={isPremiumTier}
    />
  );
};

export default EditPackFormWrapper;
