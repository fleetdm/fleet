import React, { useContext, useCallback } from "react";
import { RouteComponentProps } from "react-router";

import PATHS from "router/paths";
import labelsAPI from "services/entities/labels";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { buildQueryStringFromParams } from "utilities/url";
import { IApiError } from "interfaces/errors";

import DynamicLabelForm from "pages/labels/components/DynamicLabelForm";
import { IDynamicLabelFormData } from "pages/labels/components/DynamicLabelForm/DynamicLabelForm";
import { DUPLICATE_ENTRY_ERROR } from "../ManualLabel/ManualLabel";

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
  const { currentTeam } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const onSaveNewLabel = useCallback(
    (formData: IDynamicLabelFormData) => {
      labelsAPI
        .create(formData)
        .then((res) => {
          router.push(
            `${PATHS.MANAGE_HOSTS_LABEL(
              res.label.id
            )}?${buildQueryStringFromParams({ team_id: currentTeam?.id })}`
          );
        })
        .catch((error: { data: IApiError }) => {
          renderFlash(
            "error",
            error.data.errors[0].reason.includes("Duplicate entry")
              ? DUPLICATE_ENTRY_ERROR
              : "Couldn't add label. Please try again."
          );
        });
    },
    [renderFlash, router]
  );

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
