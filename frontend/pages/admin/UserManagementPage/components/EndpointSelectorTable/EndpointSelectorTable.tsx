import React, { useMemo, useState, useCallback } from "react";
import { useQuery } from "react-query";

import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import Checkbox from "components/forms/fields/Checkbox";
import PillBadge from "components/PillBadge";
import Spinner from "components/Spinner";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import { IApiEndpoint } from "interfaces/api_endpoint";
import apiEndpointsAPI from "services/entities/api_endpoints";

const baseClass = "endpoint-selector-table";

/** Unique key for an endpoint since there's no `id` field */
const endpointKey = (ep: IApiEndpoint) => `${ep.method} ${ep.path}`;

interface IEndpointSelectorTableProps {
  selectedEndpointKeys: string[];
  onSelectionChange: (selectedKeys: string[]) => void;
}

const PAGE_SIZE = 10;

const EndpointSelectorTable = ({
  selectedEndpointKeys,
  onSelectionChange,
}: IEndpointSelectorTableProps) => {
  const [searchQuery, setSearchQuery] = useState("");
  const [currentPage, setCurrentPage] = useState(0);

  const { data: apiEndpoints, isLoading, error } = useQuery<
    IApiEndpoint[],
    Error
  >(["api_endpoints"], () => apiEndpointsAPI.loadAll(), {
    refetchOnWindowFocus: false,
  });

  const filteredEndpoints = useMemo(() => {
    if (!apiEndpoints) return [];
    if (!searchQuery) return apiEndpoints;
    const query = searchQuery.toLowerCase();
    return apiEndpoints.filter(
      (ep) =>
        ep.display_name.toLowerCase().includes(query) ||
        ep.path.toLowerCase().includes(query)
    );
  }, [apiEndpoints, searchQuery]);

  const pageCount = Math.ceil(filteredEndpoints.length / PAGE_SIZE);
  const paginatedEndpoints = useMemo(() => {
    const start = currentPage * PAGE_SIZE;
    return filteredEndpoints.slice(start, start + PAGE_SIZE);
  }, [filteredEndpoints, currentPage]);

  const handleSearch = useCallback((value: string) => {
    setSearchQuery(value);
    setCurrentPage(0);
  }, []);

  const toggleEndpoint = useCallback(
    (key: string) => {
      const isSelected = selectedEndpointKeys.includes(key);
      if (isSelected) {
        onSelectionChange(selectedEndpointKeys.filter((k) => k !== key));
      } else {
        onSelectionChange([...selectedEndpointKeys, key]);
      }
    },
    [selectedEndpointKeys, onSelectionChange]
  );

  const renderDeprecatedBadge = () => (
    <PillBadge tipContent="This endpoint is deprecated and may be removed in a future version.">
      Deprecated
    </PillBadge>
  );

  return (
    <div className={baseClass}>
      <InputField
        name="endpoint-search"
        placeholder='Search by name (e.g. "Get host") or path (e.g. "api/v1/hosts/:id")'
        onChange={handleSearch}
        value={searchQuery}
        inputWrapperClass={`${baseClass}__search`}
      />
      {isLoading && (
        <div className={`${baseClass}__loading`}>
          <Spinner />
        </div>
      )}
      {error && (
        <div className={`${baseClass}__empty`}>
          <p>Could not load API endpoints.</p>
        </div>
      )}
      {!isLoading && !error && filteredEndpoints.length === 0 && (
        <div className={`${baseClass}__empty`}>
          <p>No matching API endpoints</p>
        </div>
      )}
      {!isLoading && !error && filteredEndpoints.length > 0 && (
        <>
          <table className={`${baseClass}__table`}>
            <thead>
              <tr>
                <th className={`${baseClass}__checkbox-col`} />
                <th>Name</th>
                <th>Protocol</th>
                <th>Path</th>
              </tr>
            </thead>
            <tbody>
              {paginatedEndpoints.map((endpoint) => {
                const key = endpointKey(endpoint);
                return (
                  <tr key={key}>
                    <td className={`${baseClass}__checkbox-col`}>
                      <Checkbox
                        value={selectedEndpointKeys.includes(key)}
                        name={`endpoint-${key}`}
                        onChange={() => toggleEndpoint(key)}
                      />
                    </td>
                    <td>
                      <TextCell value={endpoint.display_name} />
                      {endpoint.deprecated && renderDeprecatedBadge()}
                    </td>
                    <td>
                      <code>{endpoint.method}</code>
                    </td>
                    <td>
                      <code>{endpoint.path}</code>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          {pageCount > 1 && (
            <div className={`${baseClass}__pagination`}>
              <button
                type="button"
                disabled={currentPage === 0}
                onClick={() => setCurrentPage((p) => p - 1)}
              >
                Previous
              </button>
              <span>
                Page {currentPage + 1} of {pageCount}
              </span>
              <button
                type="button"
                disabled={currentPage >= pageCount - 1}
                onClick={() => setCurrentPage((p) => p + 1)}
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default EndpointSelectorTable;
