import React, { useContext } from "react";
import { useQuery } from "react-query";
import { RouteComponentProps } from "react-router";
import { AxiosError } from "axios";

import PATHS from "router/paths";
import labelsAPI, {
  IGetHostsInLabelResponse,
  IGetLabelResonse,
} from "services/entities/labels";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { ILabel } from "interfaces/label";
import { IHost } from "interfaces/host";
import { NotificationContext } from "context/notification";

import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

import DynamicLabelForm from "../components/DynamicLabelForm";
import ManualLabelForm from "../components/ManualLabelForm";
import { IDynamicLabelFormData } from "../components/DynamicLabelForm/DynamicLabelForm";
import { IManualLabelFormData } from "../components/ManualLabelForm/ManualLabelForm";

const baseClass = "edit-label-page";

interface IEditLabelPageRouteParams {
  label_id: string;
}

type IEditLabelPageProps = RouteComponentProps<
  never,
  IEditLabelPageRouteParams
>;

const EditLabelPage = ({ routeParams, router }: IEditLabelPageProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const labelId = parseInt(routeParams.label_id, 10);

  const {
    data: label,
    isLoading: isLoadingLabel,
    isError: isErrorLabel,
  } = useQuery<IGetLabelResonse, AxiosError, ILabel>(
    ["label", labelId],
    () => labelsAPI.getLabel(labelId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (data) => data.label,
    }
  );

  const {
    data: targetedHosts,
    isLoading: isLoadingHosts,
    isError: isErrorHosts,
  } = useQuery<IGetHostsInLabelResponse, AxiosError, IHost[]>(
    ["hosts"],
    () => {
      return labelsAPI.getHostsInLabel(labelId);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (data) => data.hosts,
      enabled: label?.label_membership_type === "manual",
    }
  );

  const onCancelEdit = () => {
    router.goBack();
  };

  const onUpdateLabel = async (
    formData: IDynamicLabelFormData | IManualLabelFormData
  ) => {
    try {
      const res = await labelsAPI.update(labelId, formData);
      router.push(PATHS.MANAGE_HOSTS_LABEL(res.label.id));
      renderFlash("success", "Label updated successfully.");
    } catch {
      renderFlash("error", "Couldn't edit label. Please try again.");
    }
  };

  const renderContent = () => {
    if (isLoadingLabel || isLoadingHosts) {
      return <Spinner />;
    }

    if (isErrorLabel || isErrorHosts) {
      return <DataError />;
    }

    if (!label) return null;

    if (label.label_type === "builtin") {
      return (
        <DataError
          description="Built in labels cannot be edited"
          excludeIssueLink
        />
      );
    }

    return label.label_membership_type === "dynamic" ? (
      <DynamicLabelForm
        defaultName={label.name}
        defaultDescription={label.description}
        defaultQuery={label.query}
        defaultPlatform={label.platform}
        isEditing
        onSave={onUpdateLabel}
        onCancel={onCancelEdit}
      />
    ) : (
      <ManualLabelForm
        key={targetedHosts?.toString()}
        defaultName={label.name}
        defaultDescription={label.description}
        defaultTargetedHosts={targetedHosts}
        onSave={onUpdateLabel}
        onCancel={onCancelEdit}
      />
    );
  };

  return (
    <>
      <MainContent className={baseClass}>
        <h1>Edit label</h1>
        {renderContent()}
      </MainContent>
    </>
  );
};

export default EditLabelPage;
