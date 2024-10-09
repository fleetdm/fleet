import React, { useCallback, useContext } from "react";
import { RouteComponentProps } from "react-router";

import PATHS from "router/paths";
import labelsAPI from "services/entities/labels";
import { NotificationContext } from "context/notification";
import { IApiError } from "interfaces/errors";

import ManualLabelForm from "pages/labels/components/ManualLabelForm";
import { IManualLabelFormData } from "pages/labels/components/ManualLabelForm/ManualLabelForm";

const baseClass = "manual-label";

export const DUPLICATE_ENTRY_ERROR =
  "Couldn't add. A label with this name already exists.";

type IManualLabelProps = RouteComponentProps<never, never>;

const ManualLabel = ({ router }: IManualLabelProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const onSaveNewLabel = useCallback(
    (formData: IManualLabelFormData) => {
      labelsAPI
        .create(formData)
        .then((res) => {
          router.push(PATHS.MANAGE_HOSTS_LABEL(res.label.id));
          renderFlash("success", "Label added successfully.");
        })
        .catch((error: { data: IApiError }) => {
          if (error.data.errors[0].reason.includes("Duplicate entry")) {
            renderFlash("error", DUPLICATE_ENTRY_ERROR);
          } else renderFlash("error", "Couldn't add label. Please try again.");
        });
    },
    [renderFlash, router]
  );

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
