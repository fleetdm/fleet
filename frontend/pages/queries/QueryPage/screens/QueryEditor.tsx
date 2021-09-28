import React, { useContext, useEffect } from "react";
import { Link } from "react-router";
import { useDispatch } from "react-redux";
import { UseMutateAsyncFunction } from "react-query";

import queryAPI from "services/entities/queries";
import { AppContext } from "context/app";
import { QueryContext } from "context/query"; // @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import PATHS from "router/paths"; // @ts-ignore
import debounce from "utilities/debounce"; // @ts-ignore
import deepDifference from "utilities/deep_difference";
import { IQueryFormData, IQuery } from "interfaces/query";

import QueryForm from "pages/queries/QueryPage/components/QueryForm";
import BackChevron from "../../../../../assets/images/icon-chevron-down-9x6@2x.png";

interface IQueryEditorProps {
  router: any;
  baseClass: string;
  queryIdForEdit: number | null;
  storedQueryError: any;
  showOpenSchemaActionText: boolean;
  isStoredQueryLoading: boolean;
  createQuery: UseMutateAsyncFunction<any, unknown, IQueryFormData, unknown>;
  onOsqueryTableSelect: (tableName: string) => void;
  goToSelectTargets: () => void;
  onOpenSchemaSidebar: () => void;
  renderLiveQueryWarning: () => JSX.Element | null;
}

const QueryEditor = ({
  router,
  baseClass,
  queryIdForEdit,
  storedQueryError,
  showOpenSchemaActionText,
  isStoredQueryLoading,
  createQuery,
  onOsqueryTableSelect,
  goToSelectTargets,
  onOpenSchemaSidebar,
  renderLiveQueryWarning,
}: IQueryEditorProps) => {
  const dispatch = useDispatch();
  const { currentUser } = useContext(AppContext);
  const {
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryBody,
    lastEditedQueryObserverCanRun,
  } = useContext(QueryContext);

  useEffect(() => {
    if (storedQueryError) {
      dispatch(
        renderFlash(
          "error",
          "Something went wrong retrieving your query. Please try again."
        )
      );
    }
  }, []);

  const onSaveQueryFormSubmit = debounce(async (formData: IQueryFormData) => {
    try {
      const { query }: { query: IQuery } = await createQuery(formData);
      router.push(PATHS.EDIT_QUERY(query));
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
    if (!queryIdForEdit) {
      return false;
    }

    const updatedQuery = deepDifference(formData, {
      lastEditedQueryName,
      lastEditedQueryDescription,
      lastEditedQueryBody,
      lastEditedQueryObserverCanRun,
    });

    try {
      await queryAPI.update(queryIdForEdit, updatedQuery);
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
        goToSelectTargets={goToSelectTargets}
        onOsqueryTableSelect={onOsqueryTableSelect}
        onUpdate={onUpdateQuery}
        queryIdForEdit={queryIdForEdit}
        isStoredQueryLoading={isStoredQueryLoading}
        showOpenSchemaActionText={showOpenSchemaActionText}
        onOpenSchemaSidebar={onOpenSchemaSidebar}
        renderLiveQueryWarning={renderLiveQueryWarning}
      />
    </div>
  );
};

export default QueryEditor;
