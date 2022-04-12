import React, { useContext, useEffect, useState } from "react";
import { Link } from "react-router";
import { InjectedRouter } from "react-router/lib/Router";
import { UseMutateAsyncFunction } from "react-query";

import queryAPI from "services/entities/queries";
import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import { NotificationContext } from "context/notification";
import { IQueryFormData, IQuery } from "interfaces/query";
import PATHS from "router/paths"; // @ts-ignore
import debounce from "utilities/debounce"; // @ts-ignore
import deepDifference from "utilities/deep_difference";

import QueryForm from "pages/queries/QueryPage/components/QueryForm";
import BackChevron from "../../../../../assets/images/icon-chevron-down-9x6@2x.png";

interface IQueryEditorProps {
  router: InjectedRouter;
  baseClass: string;
  queryIdForEdit: number | null;
  storedQuery: IQuery | undefined;
  storedQueryError: Error | null;
  showOpenSchemaActionText: boolean;
  isStoredQueryLoading: boolean;
  createQuery: UseMutateAsyncFunction<
    { query: IQuery },
    unknown,
    IQueryFormData,
    unknown
  >;
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
  const { currentUser } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

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
      renderFlash(
        "error",
        "Something went wrong retrieving your query. Please try again."
      );
    }
  }, []);

  const [backendValidators, setBackendValidators] = useState<{
    [key: string]: string;
  }>({});

  const onSaveQueryFormSubmit = debounce(async (formData: IQueryFormData) => {
    try {
      const { query }: { query: IQuery } = await createQuery(formData);
      router.push(PATHS.EDIT_QUERY(query));
      renderFlash("success", "Query created!");
      setBackendValidators({});
    } catch (createError: any) {
      console.error(createError);
      if (createError.data.errors[0].reason.includes("already exists")) {
        setBackendValidators({ name: "A query with this name already exists" });
      } else {
        renderFlash(
          "error",
          "Something went wrong creating your query. Please try again."
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
      renderFlash("success", "Query updated!");
    } catch (updateError: any) {
      console.error(updateError);
      if (updateError.data.errors[0].reason.includes("Duplicate")) {
        renderFlash("error", "A query with this name already exists.");
      } else {
        renderFlash(
          "error",
          "Something went wrong updating your query. Please try again."
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
        router={router}
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
        backendValidators={backendValidators}
      />
    </div>
  );
};

export default QueryEditor;
