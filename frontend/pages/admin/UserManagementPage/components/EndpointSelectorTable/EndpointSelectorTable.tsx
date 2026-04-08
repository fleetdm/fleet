import React, { useMemo, useState, useCallback } from "react";

import TextCell from "components/TableContainer/DataTable/TextCell/TextCell";
import Checkbox from "components/forms/fields/Checkbox";
import PillBadge from "components/PillBadge";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

import FLEET_API_ENDPOINTS from "./endpointsMockData";

const baseClass = "endpoint-selector-table";

interface IEndpointSelectorTableProps {
  selectedEndpointIds: string[];
  onSelectionChange: (selectedIds: string[]) => void;
}

const PAGE_SIZE = 10;

const EndpointSelectorTable = ({
  selectedEndpointIds,
  onSelectionChange,
}: IEndpointSelectorTableProps) => {
  const [searchQuery, setSearchQuery] = useState("");
  const [currentPage, setCurrentPage] = useState(0);

  const filteredEndpoints = useMemo(() => {
    if (!searchQuery) return FLEET_API_ENDPOINTS;
    const query = searchQuery.toLowerCase();
    return FLEET_API_ENDPOINTS.filter(
      (ep) =>
        ep.name.toLowerCase().includes(query) ||
        ep.path.toLowerCase().includes(query)
    );
  }, [searchQuery]);

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
    (endpointId: string) => {
      const isSelected = selectedEndpointIds.includes(endpointId);
      if (isSelected) {
        onSelectionChange(
          selectedEndpointIds.filter((id) => id !== endpointId)
        );
      } else {
        onSelectionChange([...selectedEndpointIds, endpointId]);
      }
    },
    [selectedEndpointIds, onSelectionChange]
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
      {filteredEndpoints.length === 0 ? (
        <div className={`${baseClass}__empty`}>
          <p>No matching API endpoints</p>
        </div>
      ) : (
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
              {paginatedEndpoints.map((endpoint) => (
                <tr key={endpoint.id}>
                  <td className={`${baseClass}__checkbox-col`}>
                    <Checkbox
                      value={selectedEndpointIds.includes(endpoint.id)}
                      name={`endpoint-${endpoint.id}`}
                      onChange={() => toggleEndpoint(endpoint.id)}
                    />
                  </td>
                  <td>
                    <TextCell value={endpoint.name} />
                    {endpoint.deprecated && renderDeprecatedBadge()}
                  </td>
                  <td>
                    <code>{endpoint.method}</code>
                  </td>
                  <td>
                    <code>{endpoint.path}</code>
                  </td>
                </tr>
              ))}
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
