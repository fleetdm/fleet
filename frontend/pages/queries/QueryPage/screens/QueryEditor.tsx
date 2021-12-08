import React, { useContext, useEffect } from "react";
import { Link } from "react-router";
import { useDispatch } from "react-redux";
import { InjectedRouter } from "react-router/lib/Router";
import { UseMutateAsyncFunction } from "react-query";

import queryAPI from "services/entities/queries";
import { AppContext } from "context/app";
import { QueryContext } from "context/query"; // @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import PATHS from "router/paths"; // @ts-ignore
import debounce from "utilities/debounce"; // @ts-ignore
import deepDifference from "utilities/deep_difference";
import { IQueryFormData, IQuery } from "interfaces/query";
import { IApiError } from "interfaces/errors";

import QueryForm from "pages/queries/QueryPage/components/QueryForm";
import BackChevron from "../../../../../assets/images/icon-chevron-down-9x6@2x.png";

interface IQueryEditorProps {
  router: InjectedRouter;
  baseClass: string;
  queryIdForEdit: number | null;
  storedQuery: IQuery | undefined;
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
  storedQuery,
  storedQueryError,
  showOpenSchemaActionText,
  isStoredQueryLoading,
  createQuery,
  onOsqueryTableSelect,
  goToSelectTargets,
  onOpenSchemaSidebar,
  renderLiveQueryWarning,
}: IQueryEditorProps): JSX.Element | null => {
  const dispatch = useDispatch();
  const { currentUser } = useContext(AppContext);

  // Note: The QueryContext values should always be used for any mutable query data such as query name
  // The storedQuery prop should only be used to access immutable metadata such as author id
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
    } catch (createError: any) {
      console.error(createError);
      if (createError.errors[0].reason.includes("already exists")) {
        dispatch(
          renderFlash("error", "A query with this name already exists.")
        );
      } else {
        dispatch(
          renderFlash(
            "error",
            "Something went wrong creating your query. Please try again."
          )
        );
      }
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
    } catch (updateError: any) {
      console.error(updateError);
      if (updateError.errors[0].reason.includes("Duplicate")) {
        dispatch(
          renderFlash("error", "A query with this name already exists.")
        );
      } else {
        dispatch(
          renderFlash(
            "error",
            "Something went wrong updating your query. Please try again."
          )
        );
      }
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
        storedQuery={storedQuery}
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
