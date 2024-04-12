import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { RouteComponentProps } from "react-router";
import { AxiosError } from "axios";
import { noop } from "lodash";

import labelsAPI, { IGetLabelResonse } from "services/entities/labels";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { ILabel } from "interfaces/label";
import { QueryContext } from "context/query";
import useToggleSidePanel from "hooks/useToggleSidePanel";

import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import SidePanelContent from "components/SidePanelContent";
import QuerySidePanel from "components/side_panels/QuerySidePanel";

import DynamicLabelForm from "../components/DynamicLabelForm";
import ManualLabelForm from "../components/ManualLabelForm";

const baseClass = "edit-label-page";

interface IEditLabelPageRouteParams {
  label_id: string;
}

type IEditLabelPageProps = RouteComponentProps<
  never,
  IEditLabelPageRouteParams
>;

const EditLabelPage = ({ routeParams, router }: IEditLabelPageProps) => {
  // GET LABEL
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

  const renderContent = () => {
    if (isLoadingLabel) {
      return <Spinner />;
    }

    // TODO: new empty state
    if (isErrorLabel) {
      return <DataError />;
    }

    if (label) {
      return label.label_membership_type ? (
        <DynamicLabelForm
          defaultQuery={label.query}
          defaultPlatform={label.platform}
          isEditing
          onSave={noop}
          onCancel={noop}
        />
      ) : (
        <ManualLabelForm onSave={noop} onCancel={noop} />
      );
    }

    return null;
  };

  // GET HOSTS
  // host;
  return (
    <>
      <MainContent className={baseClass}>
        <h1>Edit label</h1>
        {renderContent()}
        {/* <ManualLabelForm d={hosts} /> */}
      </MainContent>
    </>
  );
};

export default EditLabelPage;
