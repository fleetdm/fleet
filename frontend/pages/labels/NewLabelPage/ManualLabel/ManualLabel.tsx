import React, { useContext } from "react";
import { RouteComponentProps } from "react-router";

import PATHS from "router/paths";
import labelsAPI from "services/entities/labels";
import { NotificationContext } from "context/notification";

import ManualLabelForm from "pages/labels/components/ManualLabelForm";
import { IManualLabelFormData } from "pages/labels/components/ManualLabelForm/ManualLabelForm";

const baseClass = "manual-label";

type IManualLabelProps = RouteComponentProps<never, never>;

const ManualLabel = ({ router }: IManualLabelProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const onSaveNewLabel = async (formData: IManualLabelFormData) => {
    try {
      const res = await labelsAPI.create(formData);
      router.push(PATHS.MANAGE_HOSTS_LABEL(res.label.id));
      renderFlash("success", "Label added successfully.");
    } catch {
      renderFlash("error", "Couldn't add label. Please try again.");
    }
  };

  const onCancelLabel = () => {
    router.goBack();
  };

  return (
    <div className={baseClass}>
      <ManualLabelForm onSave={onSaveNewLabel} onCancel={onCancelLabel} />
    </div>
  );
};

export default ManualLabel;
