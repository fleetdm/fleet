import React, { useContext } from "react";
import { RouteComponentProps } from "react-router";

import PATHS from "router/paths";
import labelsAPI from "services/entities/labels";
import { NotificationContext } from "context/notification";

import DynamicLabelForm from "pages/labels/components/DynamicLabelForm";
import { IDynamicLabelFormData } from "pages/labels/components/DynamicLabelForm/DynamicLabelForm";

const baseClass = "dynamic-label";

const DEFAULT_QUERY = "SELECT 1 FROM os_version WHERE major >= 13;";

type IDynamicLabelProps = RouteComponentProps<never, never> & {
  showOpenSidebarButton: boolean;
  onOpenSidebar: () => void;
  onOsqueryTableSelect: (tableName: string) => void;
};

const DynamicLabel = ({
  showOpenSidebarButton,
  router,
  onOpenSidebar,
  onOsqueryTableSelect,
}: IDynamicLabelProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const onSaveNewLabel = async (formData: IDynamicLabelFormData) => {
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
      <DynamicLabelForm
        defaultQuery={DEFAULT_QUERY}
        showOpenSidebarButton={showOpenSidebarButton}
        onOpenSidebar={onOpenSidebar}
        onOsqueryTableSelect={onOsqueryTableSelect}
        onSave={onSaveNewLabel}
        onCancel={onCancelLabel}
      />
    </div>
  );
};

export default DynamicLabel;
