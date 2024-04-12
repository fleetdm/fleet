import React, { useState } from "react";
import { useQuery } from "react-query";

import { IHost } from "interfaces/host";
import targetsAPI, {
  ITargetsCountResponse,
  ITargetsSearchResponse,
} from "services/entities/targets";

import TargetsInput from "components/LiveQuery/TargetsInput";

import LabelForm from "../LabelForm";
import { ILabelFormData } from "../LabelForm/LabelForm";
import { Row } from "react-table";
import { generateTableHeaders } from "./LabelHostTargetTableConfig";

const baseClass = "ManualLabelForm";

export interface IManualLabelFormData {
  name: string;
  description: string;
  hosts: string[];
}

interface ITargetsQueryKey {
  scope: string;
  query?: string | null;
  excludedHostIds?: number[];
}

interface IManualLabelFormProps {
  defaultSelectedHosts?: IHost[];
  onSave: (formData: IManualLabelFormData) => void;
  onCancel: () => void;
}

const ManualLabelForm = ({
  defaultSelectedHosts = [],
  onSave,
  onCancel,
}: IManualLabelFormProps) => {
  const [searchQuery, setSearchQuery] = useState<string>("");
  const [selectedHosts, setSelectedHosts] = useState<IHost[]>(
    defaultSelectedHosts
  );

  const {
    data: hostTargets,
    isFetching: isFetchingSearchResults,
    error: errorSearchResults,
  } = useQuery<ITargetsSearchResponse, Error, IHost[], ITargetsQueryKey[]>(
    [
      {
        scope: "labels-targets-search",
        query: searchQuery,
        excludedHostIds: [], // TODO: add this
      },
    ],
    ({ queryKey }) => {
      const { query, excludedHostIds } = queryKey[0];
      return targetsAPI.search({
        query: query ?? "",
        excluded_host_ids: excludedHostIds ?? null,
      });
    },
    {
      select: (data) => data.hosts,
      enabled: searchQuery !== "",
    }
  );

  const onHostSelect = (row: Row<IHost>) => {
    setSelectedHosts((prevHosts) => prevHosts.concat(row.original));
    setSearchQuery("");
  };

  const onHostRemove = (row: Row<IHost>) => {
    setSelectedHosts((prevHosts) =>
      prevHosts.filter((h) => h.id !== row.original.id)
    );
  };

  const onSaveNewLabel = (
    formData: ILabelFormData,
    labelFormDataValid: boolean
  ) => {
    console.log("data", formData);
  };

  const tableConfig = generateTableHeaders(onHostRemove);

  return (
    <div className={baseClass}>
      <LabelForm
        onCancel={onCancel}
        onSave={onSaveNewLabel}
        additionalFields={
          <TargetsInput
            searchText={searchQuery}
            setSearchText={setSearchQuery}
            tableColumnConifg={tableConfig}
            isTargetsLoading={false}
            hasFetchError={false}
            searchResults={hostTargets ?? []}
            targetedHosts={selectedHosts}
            handleRowSelect={onHostSelect}
          />
        }
      />
    </div>
  );
};

export default ManualLabelForm;
