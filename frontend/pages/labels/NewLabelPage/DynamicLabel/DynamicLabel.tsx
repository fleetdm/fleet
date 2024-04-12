import React from "react";
import { RouteComponentProps } from "react-router";

import DynamicLabelForm from "pages/labels/components/DynamicLabelForm";
import { IDynamicLabelFormData } from "pages/labels/components/DynamicLabelForm/DynamicLabelForm";

const baseClass = "dynamic-label";

type IDynamicLabelProps = RouteComponentProps<never, never> & {
  showOpenSidebarButton: boolean;
  onOpenSidebar: () => void;
};

const DynamicLabel = ({
  showOpenSidebarButton,
  router,
  onOpenSidebar,
}: IDynamicLabelProps) => {
  const onSaveNewLabel = (formData: IDynamicLabelFormData) => {
    console.log("data", formData);
  };

  const onCancelLabel = () => {
    router.goBack();
  };

  return (
    <div className={baseClass}>
      <DynamicLabelForm
        showOpenSidebarButton={showOpenSidebarButton}
        onOpenSidebar={onOpenSidebar}
        onSave={onSaveNewLabel}
        onCancel={onCancelLabel}
      />
    </div>
  );
};

export default DynamicLabel;
