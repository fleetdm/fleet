import React, { useState, useContext, useEffect } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { RouteProps } from "react-router/lib/Route";

import PATHS from "router/paths";
import MainContent from "components/MainContent";
import SidePanelContent from "components/SidePanelContent";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import { QueryContext } from "context/query";
import { NotificationContext } from "context/notification";
import { IApiError } from "interfaces/errors";
import { ILabel, ILabelFormData } from "interfaces/label";
import labelsAPI, { ILabelsResponse } from "services/entities/labels";
import deepDifference from "utilities/deep_difference";
import LabelForm from "pages/hosts/ManageHostsPage/components/LabelForm";

const baseClass = "labels";

interface ILabelPageProps {
  route: RouteProps;
  router: InjectedRouter;
  params: Params;
  location: {
    pathname: string;
  };
}

const DEFAULT_CREATE_LABEL_ERRORS = {
  name: "",
};

const LabelPage = ({
  route,
  router,
  params,
  location,
}: ILabelPageProps): JSX.Element | null => {
  const isEditLabel = !location.pathname.includes("new");

  const [selectedLabel, setSelectedLabel] = useState<ILabel>();
  const [isSidebarOpen, setIsSidebarOpen] = useState<boolean>(true);
  const [
    showOpenSchemaActionText,
    setShowOpenSchemaActionText,
  ] = useState<boolean>(false);
  const [labelValidator, setLabelValidator] = useState<{
    [key: string]: string;
  }>(DEFAULT_CREATE_LABEL_ERRORS);
  const { selectedOsqueryTable, setSelectedOsqueryTable } = useContext(
    QueryContext
  );

  const { renderFlash } = useContext(NotificationContext);

  const showSidebar = isSidebarOpen;

  const { data: labels, error: labelsError } = useQuery<
    ILabelsResponse,
    Error,
    ILabel[]
  >(["labels"], () => labelsAPI.loadAll(), {
    select: (data: ILabelsResponse) => data.labels,
  });

  useEffect(() => {
    setShowOpenSchemaActionText(!isSidebarOpen);
  }, [isSidebarOpen]);

  const onCloseSchemaSidebar = () => {
    setIsSidebarOpen(false);
  };

  const onOpenSchemaSidebar = () => {
    setIsSidebarOpen(true);
  };

  const onOsqueryTableSelect = (tableName: string) => {
    setSelectedOsqueryTable(tableName);
  };

  const onEditLabel = (formData: ILabelFormData) => {
    if (!selectedLabel) {
      console.error("Label isn't available. This should not happen.");
      return;
    }

    const updateAttrs = deepDifference(formData, selectedLabel);

    labelsAPI
      .update(selectedLabel, updateAttrs)
      .then(() => {
        router.push(PATHS.MANAGE_HOSTS);
        renderFlash(
          "success",
          "Label updated. Try refreshing this page in just a moment to see the updated host count for your label."
        );
        setLabelValidator({});
      })
      .catch((updateError: { data: IApiError }) => {
        if (updateError.data.errors[0].reason.includes("Duplicate")) {
          setLabelValidator({
            name: "A label with this name already exists",
          });
        } else {
          renderFlash("error", "Could not create label. Please try again.");
        }
      });
  };

  const onAddLabel = (formData: ILabelFormData) => {
    labelsAPI
      .create(formData)
      .then(() => {
        router.push(PATHS.MANAGE_HOSTS);
        renderFlash(
          "success",
          "Label created. Try refreshing this page in just a moment to see the updated host count for your label."
        );
        setLabelValidator({});
      })
      .catch((updateError: any) => {
        if (updateError.data.errors[0].reason.includes("Duplicate")) {
          setLabelValidator({
            name: "A label with this name already exists",
          });
        } else {
          renderFlash("error", "Could not create label. Please try again.");
        }
      });
  };

  const onCancelLabel = () => {
    router.goBack();
  };

  return (
    <>
      <MainContent className={baseClass}>
        <div className={`${baseClass}__wrapper`}>
          <LabelForm
            selectedLabel={selectedLabel}
            onCancel={onCancelLabel}
            isEdit={isEditLabel}
            handleSubmit={isEditLabel ? onEditLabel : onAddLabel}
            onOpenSchemaSidebar={onOpenSchemaSidebar}
            onOsqueryTableSelect={onOsqueryTableSelect}
            baseError={labelsError?.message || ""}
            backendValidators={labelValidator}
            showOpenSchemaActionText={showOpenSchemaActionText}
          />
        </div>
      </MainContent>
      {showSidebar && !isEditLabel && (
        <SidePanelContent>
          <QuerySidePanel
            key="query-side-panel"
            onOsqueryTableSelect={onOsqueryTableSelect}
            selectedOsqueryTable={selectedOsqueryTable}
            onClose={onCloseSchemaSidebar}
          />
        </SidePanelContent>
      )}
    </>
  );
};

export default LabelPage;
