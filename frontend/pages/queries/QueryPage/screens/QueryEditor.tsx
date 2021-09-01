import React from "react";
import { Dispatch } from "redux";
import { Link } from "react-router";
import { push } from "react-router-redux";
import { UseMutateAsyncFunction } from "react-query";

import queryAPI from "services/entities/queries"; // @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import PATHS from "router/paths"; // @ts-ignore
import debounce from "utilities/debounce"; // @ts-ignore
import deepDifference from "utilities/deep_difference";
import { IUser } from "interfaces/user";
import { IQueryFormData, IQuery } from "interfaces/query";

import QueryForm from "components/forms/queries/QueryForm";
import { hasSavePermissions } from "pages/queries/QueryPage/helpers";
import BackChevron from "../../../../../assets/images/icon-chevron-down-9x6@2x.png";

interface IQueryEditorProps {
  baseClass: string;
  currentUser: IUser | null;
  storedQuery: IQuery | undefined;
  error: any;
  showOpenSchemaActionText: boolean;
  isStoredQueryLoading: boolean;
  dispatch: Dispatch;
  createQuery: UseMutateAsyncFunction<any, unknown, IQueryFormData, unknown>;
  onOsqueryTableSelect: (tableName: string) => void;
  goToSelectTargets: () => void;
  setTypedQueryBody: (value: string) => void;
  onOpenSchemaSidebar: () => void;
  renderLiveQueryWarning: () => JSX.Element | null;
}

const QueryEditor = ({
  baseClass,
  currentUser,
  storedQuery,
  error,
  showOpenSchemaActionText,
  isStoredQueryLoading,
  createQuery,
  onOsqueryTableSelect,
  goToSelectTargets,
  setTypedQueryBody,
  dispatch,
  onOpenSchemaSidebar,
  renderLiveQueryWarning,
}: IQueryEditorProps) => {
  const onSaveQueryFormSubmit = debounce(async (formData: IQueryFormData) => {
    try {
      const { query }: { query: IQuery } = await createQuery(formData);
      dispatch(push(PATHS.EDIT_QUERY(query)));
      dispatch(renderFlash("success", "Query created!"));
    } catch (createError) {
      console.error(createError);
      dispatch(
        renderFlash(
          "error",
          "Something went wrong creating your query. Please try again."
        )
      );
    }
  });

  const onUpdateQuery = async (formData: IQueryFormData) => {
    if (!storedQuery) {
      return false;
    }

    const updatedQuery = deepDifference(formData, storedQuery);

    try {
      await queryAPI.update(storedQuery, updatedQuery);
      dispatch(renderFlash("success", "Query updated!"));
    } catch (updateError) {
      console.error(updateError);
      dispatch(
        renderFlash(
          "error",
          "Something went wrong updating your query. Please try again."
        )
      );
    }

    return false;
  };

  const onChangeQueryFormField = (fieldName: string, value: string) => {
    if (fieldName === "query") {
      setTypedQueryBody(value);
    }

    return false;
  };

  if (!currentUser) {
    return null;
  }

  return (
    <div className={`${baseClass}__form body-wrap`}>
      <Link to={PATHS.MANAGE_QUERIES} className={`${baseClass}__back-link`}>
        <img src={BackChevron} alt="back chevron" id="back-chevron" />
        <span>Back to queries</span>
      </Link>
      <QueryForm
        onCreateQuery={onSaveQueryFormSubmit}
        onChangeFunc={onChangeQueryFormField}
        goToSelectTargets={goToSelectTargets}
        onOsqueryTableSelect={onOsqueryTableSelect}
        onUpdate={onUpdateQuery}
        serverErrors={error || {}}
        storedQuery={storedQuery}
        isStoredQueryLoading={isStoredQueryLoading}
        hasSavePermissions={hasSavePermissions(currentUser)}
        showOpenSchemaActionText={showOpenSchemaActionText}
        onOpenSchemaSidebar={onOpenSchemaSidebar}
        renderLiveQueryWarning={renderLiveQueryWarning}
      />
    </div>
  );
};

export default QueryEditor;
