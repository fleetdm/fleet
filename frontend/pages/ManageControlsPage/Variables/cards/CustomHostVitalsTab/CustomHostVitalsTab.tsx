import React, { useCallback, useContext, useMemo, useState } from "react";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import useGitOpsMode from "hooks/useGitOpsMode";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getNextLocationPath } from "utilities/helpers";
import { ICustomHostVital } from "interfaces/custom_host_vitals";
import { IListCustomHostVitalsResponse } from "services/entities/custom_host_vitals";
// TODO(#48559): replace mock with live API — swap for
// "services/entities/custom_host_vitals" once CRUD endpoints exist.
import customHostVitalsAPI from "services/entities/custom_host_vitals_mock";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Spinner from "components/Spinner";
import EmptyState from "components/EmptyState";
import PageDescription from "components/PageDescription";
import SectionHeader from "components/SectionHeader";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TableContainer from "components/TableContainer";
import { ITableQueryData } from "components/TableContainer/TableContainer";
import TableCount from "components/TableContainer/TableCount";

import generateTableHeaders from "./CustomHostVitalsTableConfig";
import { IVariablesCardProps } from "../../VariablesNavItems";
import AddCustomHostVitalModal from "../../components/AddCustomHostVitalModal";
import EditCustomHostVitalModal from "../../components/EditCustomHostVitalModal";
import DeleteCustomHostVitalModal from "../../components/DeleteCustomHostVitalModal";

const baseClass = "custom-host-vitals-tab";

const CustomHostVitalsTab: React.FC<IVariablesCardProps> = ({
  router,
  location,
}) => {
  const { isGlobalAdmin, isGlobalMaintainer } = useContext(AppContext);
  const { gitOpsModeEnabled } = useGitOpsMode();

  const canEdit = isGlobalAdmin || isGlobalMaintainer;

  const searchQuery = location.query.query ?? "";
  const [showAddModal, setShowAddModal] = useState(false);
  const [vitalToEdit, setVitalToEdit] = useState<
    ICustomHostVital | undefined
  >();
  const [vitalToDelete, setVitalToDelete] = useState<
    ICustomHostVital | undefined
  >();

  const apiParams = { query: searchQuery };
  const { data, isLoading, isFetching, refetch } = useQuery<
    IListCustomHostVitalsResponse,
    Error,
    IListCustomHostVitalsResponse
  >(
    ["customHostVitals", apiParams],
    () => customHostVitalsAPI.getCustomHostVitals(apiParams),
    // keepPreviousData keeps `data` populated across search-driven key changes
    // so TableContainer (and its search box) doesn't unmount/remount empty.
    { ...DEFAULT_USE_QUERY_OPTIONS, keepPreviousData: true }
  );

  const vitals = useMemo(() => data?.custom_host_vitals ?? [], [data]);
  const count = data?.count ?? 0;

  const onQueryChange = useCallback(
    (queryData: ITableQueryData) => {
      const nextSearchQuery = queryData.searchQuery;

      if (nextSearchQuery === searchQuery) {
        return;
      }

      router.replace(
        getNextLocationPath({
          pathPrefix: PATHS.CONTROLS_VARIABLES_CUSTOM_HOST_VITALS,
          queryParams: {
            ...location.query,
            query: nextSearchQuery || undefined,
          },
        })
      );
    },
    [searchQuery, router, location.query]
  );

  const onClickAdd = () => setShowAddModal(true);

  const onSaveAdd = () => {
    setShowAddModal(false);
    refetch();
  };

  const onSaveEdit = () => {
    setVitalToEdit(undefined);
    refetch();
  };

  const onDeleted = () => {
    setVitalToDelete(undefined);
    refetch();
  };

  const tableHeaders = useMemo(
    () =>
      generateTableHeaders({
        canEdit: !!canEdit,
        gitOpsModeEnabled,
        onEdit: setVitalToEdit,
        onDelete: setVitalToDelete,
      }),
    [canEdit, gitOpsModeEnabled, setVitalToEdit, setVitalToDelete]
  );

  const renderCount = useCallback(
    () => <TableCount name="vitals" count={count} />,
    [count]
  );

  const isSearching = searchQuery !== "";
  // "No vitals at all" is distinct from "no vitals match the search". The
  // former is only known when the unfiltered list is empty, so treat any active
  // search as the search-empty case.
  const isEmpty = !isLoading && count === 0 && !isSearching;

  const renderAddButton = (variant: "inverse" | "default") =>
    canEdit ? (
      <GitOpsModeTooltipWrapper
        renderChildren={(disableChildren) => (
          <Button
            variant={variant === "inverse" ? "inverse" : undefined}
            size={variant === "inverse" ? "small" : undefined}
            onClick={onClickAdd}
            disabled={disableChildren}
          >
            {variant === "inverse" && <Icon name="plus" />}
            <span>Add vital</span>
          </Button>
        )}
      />
    ) : undefined;

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
          header="No custom host vitals"
          info={
            canEdit
              ? "Add new vitals to display custom values and access them as variables."
              : "No custom host vitals have been added."
          }
          primaryButton={renderAddButton("default")}
        />
      );
    }

    return (
      <TableContainer
        columnConfigs={tableHeaders}
        data={vitals}
        isLoading={isFetching}
        defaultSortHeader="name"
        defaultSortDirection="asc"
        defaultSearchQuery={searchQuery}
        inputPlaceHolder="Search by name"
        onQueryChange={onQueryChange}
        emptyComponent={() => (
          <EmptyState
            header="No matching custom host vitals"
            info="No custom host vitals match those filters."
          />
        )}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        searchable
        renderCount={renderCount}
        isClientSidePagination
      />
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader
        title="Custom host vitals"
        alignLeftHeaderVertically
        details={renderAddButton("inverse")}
      />
      <PageDescription
        variant="tab-panel"
        content="Manage custom fields on hosts. Their values can be set manually on each host's details page, or via API integration."
      />
      {renderContent()}
      {showAddModal && (
        <AddCustomHostVitalModal
          onCancel={() => setShowAddModal(false)}
          onSave={onSaveAdd}
        />
      )}
      {vitalToEdit && (
        <EditCustomHostVitalModal
          vital={vitalToEdit}
          onCancel={() => setVitalToEdit(undefined)}
          onSave={onSaveEdit}
        />
      )}
      {vitalToDelete && (
        <DeleteCustomHostVitalModal
          vital={vitalToDelete}
          onExit={() => setVitalToDelete(undefined)}
          onDelete={onDeleted}
        />
      )}
    </div>
  );
};

export default CustomHostVitalsTab;
