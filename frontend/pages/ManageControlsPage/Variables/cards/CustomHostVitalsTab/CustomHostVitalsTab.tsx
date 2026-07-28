import React, { useCallback, useContext, useMemo, useState } from "react";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getNextLocationPath } from "utilities/helpers";
import { ICustomHostVital } from "interfaces/custom_host_vitals";
import customHostVitalsAPI, {
  IListCustomHostVitalsResponse,
} from "services/entities/custom_host_vitals";

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

export const CUSTOM_HOST_VITALS_PAGE_SIZE = 20;

const CustomHostVitalsTab: React.FC<IVariablesCardProps> = ({
  router,
  location,
}) => {
  const { isGlobalAdmin, isGlobalMaintainer } = useContext(AppContext);

  const canEdit = isGlobalAdmin || isGlobalMaintainer;

  const searchQuery = location.query.query ?? "";
  const parsedPage = parseInt(location.query.page ?? "", 10);
  const pageNumber =
    Number.isNaN(parsedPage) || parsedPage < 0 ? 0 : parsedPage;
  const sortHeader = location.query.order_key || "name";
  const sortDirection: "asc" | "desc" =
    location.query.order_direction === "desc" ? "desc" : "asc";

  const [showAddModal, setShowAddModal] = useState(false);
  const [vitalToEdit, setVitalToEdit] = useState<
    ICustomHostVital | undefined
  >();
  const [vitalToDelete, setVitalToDelete] = useState<
    ICustomHostVital | undefined
  >();

  const apiParams = {
    query: searchQuery,
    page: pageNumber,
    per_page: CUSTOM_HOST_VITALS_PAGE_SIZE,
    order_key: sortHeader,
    order_direction: sortDirection,
  };
  const { data, isLoading, isFetching, refetch } = useQuery<
    IListCustomHostVitalsResponse,
    Error,
    IListCustomHostVitalsResponse
  >(
    ["customHostVitals", apiParams],
    () => customHostVitalsAPI.getCustomHostVitals(apiParams),
    // keepPreviousData keeps `data` populated across page/search key changes
    // so TableContainer (and its search box) doesn't unmount/remount empty.
    { ...DEFAULT_USE_QUERY_OPTIONS, keepPreviousData: true }
  );

  const vitals = useMemo(() => data?.custom_host_vitals ?? [], [data]);
  const count = data?.count ?? 0;

  const onQueryChange = useCallback(
    (queryData: ITableQueryData) => {
      const {
        searchQuery: nextSearchQuery,
        pageIndex,
        sortHeader: nextSortHeader,
        sortDirection: nextSortDirection,
      } = queryData;

      const searchChanged = nextSearchQuery !== searchQuery;
      const nextPage = searchChanged ? 0 : pageIndex;
      const nextOrderKey = nextSortHeader || "name";
      const nextOrderDirection = nextSortDirection === "desc" ? "desc" : "asc";

      if (
        !searchChanged &&
        nextPage === pageNumber &&
        nextOrderKey === sortHeader &&
        nextOrderDirection === sortDirection
      ) {
        return;
      }

      router.replace(
        getNextLocationPath({
          pathPrefix: PATHS.CONTROLS_VARIABLES_CUSTOM_HOST_VITALS,
          queryParams: {
            ...location.query,
            query: nextSearchQuery || undefined,
            page: nextPage || undefined,
            order_key: nextOrderKey !== "name" ? nextOrderKey : undefined,
            order_direction:
              nextOrderDirection !== "asc" ? nextOrderDirection : undefined,
          },
        })
      );
    },
    [searchQuery, pageNumber, sortHeader, sortDirection, router, location.query]
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
        onEdit: setVitalToEdit,
        onDelete: setVitalToDelete,
      }),
    [canEdit, setVitalToEdit, setVitalToDelete]
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

  const renderAddButton = (variant: "secondary" | "default") =>
    canEdit ? (
      <GitOpsModeTooltipWrapper
        renderChildren={(disableChildren) => (
          <Button
            variant={variant === "secondary" ? "secondary" : undefined}
            size={variant === "secondary" ? "small" : undefined}
            onClick={onClickAdd}
            disabled={disableChildren}
          >
            {variant === "secondary" && <Icon name="plus" />}
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
        manualSortBy
        pageIndex={pageNumber}
        pageSize={CUSTOM_HOST_VITALS_PAGE_SIZE}
        disableNextPage={
          (pageNumber + 1) * CUSTOM_HOST_VITALS_PAGE_SIZE >= count
        }
      />
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader
        title="Custom host vitals"
        alignLeftHeaderVertically
        details={renderAddButton("secondary")}
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
