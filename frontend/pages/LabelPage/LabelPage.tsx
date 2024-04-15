import React, { useState, useContext } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";

import PATHS from "router/paths";
import MainContent from "components/MainContent";
import SidePanelContent from "components/SidePanelContent";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import Spinner from "components/Spinner";
import { QueryContext } from "context/query";
import { NotificationContext } from "context/notification";
import { IApiError } from "interfaces/errors";
import { ILabel, ILabelFormData } from "interfaces/label";
import labelsAPI, { ILabelsResponse } from "services/entities/labels";
import deepDifference from "utilities/deep_difference";
import useToggleSidePanel from "hooks/useToggleSidePanel";

import LabelForm from "./LabelForm";

const baseClass = "label-page";

interface ILabelPageProps {
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
  router,
  params,
  location,
}: ILabelPageProps): JSX.Element | null => {
  const isEditLabel = !location.pathname.includes("new");

  const [selectedLabel, setSelectedLabel] = useState<ILabel>();
  const { isSidePanelOpen, setSidePanelOpen } = useToggleSidePanel(true);
  const [showOpenSchemaActionText, setShowOpenSchemaActionText] = useState(
    false
  );
  const [labelValidator, setLabelValidator] = useState<{
    [key: string]: string;
  }>(DEFAULT_CREATE_LABEL_ERRORS);
  const [isUpdatingLabel, setIsUpdatingLabel] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  const { selectedOsqueryTable, setSelectedOsqueryTable } = useContext(
    QueryContext
  );
  const { renderFlash } = useContext(NotificationContext);

  const { error: labelsError } = useQuery<ILabelsResponse, Error, ILabel[]>(
    ["labels"],
    () => labelsAPI.loadAll(),
    {
      select: (data: ILabelsResponse) => data.labels,
      onSuccess: (responseLabels: ILabel[]) => {
        if (params.label_id) {
          const selectLabel = responseLabels.find(
            (label) => label.id === parseInt(params.label_id, 10)
          );
          setSelectedLabel(selectLabel);
          setIsLoading(false);
        }
      },
    }
  );

  const onCloseSchemaSidebar = () => {
    setSidePanelOpen(false);
    setShowOpenSchemaActionText(true);
  };

  const onOpenSchemaSidebar = () => {
    setSidePanelOpen(true);
    setShowOpenSchemaActionText(false);
  };

  const onOsqueryTableSelect = (tableName: string) => {
    setSelectedOsqueryTable(tableName);
  };

  const onEditLabel = (formData: ILabelFormData) => {
    if (!selectedLabel) {
      console.error("Label isn't available. This should not happen.");
      return;
    }

    setIsUpdatingLabel(true);
    const updateAttrs = deepDifference(formData, selectedLabel);

    labelsAPI
      .update(selectedLabel, updateAttrs)
      .then(() => {
        router.push(PATHS.MANAGE_HOSTS_LABEL(selectedLabel.id));
        renderFlash(
          "success",
          "Label updated. Try refreshing this page in just a moment to see the updated host count for your label."
        );
      })
      .catch((updateError: { data: IApiError }) => {
        if (updateError.data.errors[0].reason.includes("Duplicate")) {
          setLabelValidator({
            name: "A label with this name already exists",
          });
        } else if (updateError.data.errors[0].reason.includes("built-in")) {
          setLabelValidator({
            name: "A built-in label with this name already exists",
          });
        } else if (
          updateError.data.errors[0].reason.includes(
            "Data too long for column 'name'"
          )
        ) {
          setLabelValidator({
            name: "Label name is too long",
          });
        } else if (
          updateError.data.errors[0].reason.includes(
            "Data too long for column 'description'"
          )
        ) {
          setLabelValidator({
            description: "Label description is too long",
          });
        } else {
          renderFlash("error", "Could not create label. Please try again.");
        }
      })
      .finally(() => {
        setIsUpdatingLabel(false);
      });
  };

  const onAddLabel = (formData: ILabelFormData) => {
    setIsUpdatingLabel(true);

    labelsAPI
      .create(formData)
      .then((label: ILabel) => {
        router.push(PATHS.MANAGE_HOSTS_LABEL(label.id));
        renderFlash(
          "success",
          "Label created. Try refreshing this page in just a moment to see the updated host count for your label."
        );
      })
      .catch((updateError: { data: IApiError }) => {
        if (updateError.data.errors[0].reason.includes("Duplicate")) {
          setLabelValidator({
            name: "A label with this name already exists",
          });
        } else if (updateError.data.errors[0].reason.includes("built-in")) {
          setLabelValidator({
            name: "A built-in label with this name already exists",
          });
        } else if (
          updateError.data.errors[0].reason.includes(
            "Data too long for column 'name'"
          )
        ) {
          setLabelValidator({
            name: "Label name is too long",
          });
        } else if (
          updateError.data.errors[0].reason.includes(
            "Data too long for column 'description'"
          )
        ) {
          setLabelValidator({
            description: "Label description is too long",
          });
        } else {
          renderFlash("error", "Could not create label. Please try again.");
        }
      })
      .finally(() => {
        setIsUpdatingLabel(false);
      });
  };

  const onCancelLabel = () => {
    router.goBack();
  };

  return (
    <>
      <MainContent className={baseClass}>
        <div className={`${baseClass}__wrapper`}>
          {isLoading ? (
            <Spinner />
          ) : (
            <LabelForm
              selectedLabel={selectedLabel}
              onCancel={onCancelLabel}
              isEdit={isEditLabel}
              isUpdatingLabel={isUpdatingLabel}
              handleSubmit={isEditLabel ? onEditLabel : onAddLabel}
              onOpenSchemaSidebar={onOpenSchemaSidebar}
              onOsqueryTableSelect={onOsqueryTableSelect}
              baseError={labelsError?.message || ""}
              backendValidators={labelValidator}
              showOpenSchemaActionText={showOpenSchemaActionText}
            />
          )}
        </div>
      </MainContent>
      {isSidePanelOpen && !isEditLabel && (
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
