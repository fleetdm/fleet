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

import QueryForm from "components/forms/queries/QueryForm1";
import {
  hasSavePermissions,
  selectHosts,
} from "pages/queries/QueryPage1/helpers";
import BackChevron from "../../../../../../assets/images/icon-chevron-down-9x6@2x.png";

interface IQueryEditorProps {
  baseClass: string;
  currentUser: IUser;
  dispatch: Dispatch;
  storedQuery: IQuery | undefined;
  createQuery: UseMutateAsyncFunction<any, unknown, IQueryFormData, unknown>;
  error: any;
  onOsqueryTableSelect: (tableName: string) => void;
  goToSelectTargets: () => void;
  setTypedQueryBody: (value: string) => void;
};

const QueryEditor = ({
  baseClass,
  currentUser,
  dispatch,
  storedQuery,
  createQuery,
  error,
  onOsqueryTableSelect,
  goToSelectTargets,
  setTypedQueryBody,
}: IQueryEditorProps) => {
  const onSaveQueryFormSubmit = debounce(async (formData: IQueryFormData) => {
    try {
      const { query }: { query: IQuery } = await createQuery(formData);
      dispatch(push(PATHS.EDIT_QUERY(query)));
      dispatch(renderFlash("success", "Query created!"));
    } catch (createError) {
      console.log(createError);
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
      console.log(updateError);
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
        title={storedQuery?.name || "New query"}
        hasSavePermissions={hasSavePermissions(currentUser)}
      />
    </div>
  );
};

export default QueryEditor;