import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import variablesAPI, {
  IListVariablesResponse,
} from "services/entities/variables";
import { IVariable } from "interfaces/variables";

import { AppContext } from "context/app";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import SectionHeader from "components/SectionHeader";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import EmptyState from "components/EmptyState";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon";
import PageDescription from "components/PageDescription";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";

import generateTableHeaders from "./GlobalVariablesTableConfig";
import AddCustomVariableModal from "../../components/AddCustomVariableModal";
import DeleteCustomVariableModal from "../../components/DeleteCustomVariableModal";

const baseClass = "global-variables";

export const VARIABLES_PAGE_SIZE = 20;

export interface IGlobalVariablesProps {
  router: InjectedRouter;
  location: {
    pathname: string;
    query: { add_variable?: string };
  };
}

const GlobalVariables = ({ router, location }: IGlobalVariablesProps) => {
  const { isGlobalAdmin, isGlobalMaintainer } = useContext(AppContext);

  const canEdit = isGlobalAdmin || isGlobalMaintainer;

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [variableToDelete, setVariableToDelete] = useState<
    IVariable | undefined
  >();
  const [showAddModal, setShowAddModal] = useState(false);
  const [pageNumber, setPageNumber] = useState(0);

  const apiParams = { page: pageNumber, per_page: VARIABLES_PAGE_SIZE };
  const { data, isLoading, isFetching, refetch } = useQuery<
    IListVariablesResponse,
    Error,
    IListVariablesResponse
  >(["variables", apiParams], () => variablesAPI.getVariables(apiParams), {
    // keepPreviousData keeps the current page visible while the next page
    // loads, so the table doesn't flip to a spinner between page changes.
    ...DEFAULT_USE_QUERY_OPTIONS,
    keepPreviousData: true,
  });

  const variables = useMemo(() => data?.custom_variables ?? [], [data]);
  const count = data?.count ?? 0;

  // Open the Add variable modal via deep-link (e.g. from the command
  // palette). Gate on the same predicate the in-page button uses — the
  // param must not bypass admin/maintainer-only authoring. Strip the
  // param either way so refreshes don't keep trying.
  useEffect(() => {
    if (location.query.add_variable !== "1") return;
    if (canEdit) {
      setShowAddModal(true);
    }
    const { add_variable, ...rest } = location.query;
    router.replace({ pathname: location.pathname, query: rest });
  }, [location.query, location.pathname, router, canEdit]);

  const onClickAddVariable = () => {
    setShowAddModal(true);
  };

  const onSaveVariable = () => {
    setShowAddModal(false);
    refetch();
  };

  const onDeleteVariable = () => {
    setShowDeleteModal(false);
    refetch();
  };

  const onClickDeleteVariable = useCallback((variable: IVariable) => {
    setVariableToDelete(variable);
    setShowDeleteModal(true);
  }, []);

  const onQueryChange = useCallback((newTableQuery: ITableQueryData) => {
    setPageNumber(newTableQuery.pageIndex);
  }, []);

  const tableHeaders = useMemo(
    () =>
      generateTableHeaders({
        canEdit: !!canEdit,
        onDelete: onClickDeleteVariable,
      }),
    [canEdit, onClickDeleteVariable]
  );

  const renderCount = useCallback(
    () => <TableCount name="variables" count={count} />,
    [count]
  );

  const isEmpty = !isLoading && count === 0;

  const renderContent = () => {
    if (isLoading) {
      return (
        <div className={`${baseClass}__loading`}>
          <Spinner />
        </div>
      );
    }

    if (isEmpty) {
      return (
        <EmptyState
          variant="header-list"
          header="No custom variables"
          info={
            canEdit
              ? "Add a custom variable to make it available in scripts and profiles."
              : "No custom variables are available for scripts and profiles."
          }
          primaryButton={
            canEdit ? (
              <GitOpsModeTooltipWrapper
                renderChildren={(disableChildren) => (
                  <Button
                    onClick={onClickAddVariable}
                    disabled={disableChildren}
                  >
                    Add variable
                  </Button>
                )}
              />
            ) : undefined
          }
        />
      );
    }

    return (
      <TableContainer
        columnConfigs={tableHeaders}
        data={variables}
        isLoading={isFetching}
        defaultSortHeader="name"
        defaultSortDirection="asc"
        emptyComponent={() => <EmptyState header="No custom variables" />}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        renderCount={renderCount}
        onQueryChange={onQueryChange}
        pageIndex={pageNumber}
        pageSize={VARIABLES_PAGE_SIZE}
        disableNextPage={(pageNumber + 1) * VARIABLES_PAGE_SIZE >= count}
      />
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader
        title="Global variables"
        alignLeftHeaderVertically
        details={
          canEdit ? (
            <GitOpsModeTooltipWrapper
              renderChildren={(disableChildren) => (
                <Button
                  variant="inverse"
                  size="small"
                  onClick={onClickAddVariable}
                  disabled={disableChildren}
                >
                  <Icon name="plus" />
                  <span>Add variable</span>
                </Button>
              )}
            />
          ) : undefined
        }
      />
      <PageDescription
        variant="tab-panel"
        content="Manage one-off variables that reference the same value across all hosts."
      />
      {renderContent()}
      {showAddModal && (
        <AddCustomVariableModal
          onCancel={() => setShowAddModal(false)}
          onSave={onSaveVariable}
        />
      )}
      {showDeleteModal && (
        <DeleteCustomVariableModal
          variable={variableToDelete}
          onExit={() => setShowDeleteModal(false)}
          onDeleteVariable={onDeleteVariable}
        />
      )}
    </div>
  );
};

export default GlobalVariables;
