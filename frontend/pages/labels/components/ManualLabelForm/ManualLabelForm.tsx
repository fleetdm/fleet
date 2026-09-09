import React, { useEffect, useState } from "react";
import { useQuery } from "react-query";
import { Row } from "react-table";
import { useDebouncedCallback } from "use-debounce";

import { IHost } from "interfaces/host";
import targetsAPI, { ITargetsSearchResponse } from "services/entities/targets";
import useGitOpsMode from "hooks/useGitOpsMode";

import CustomLink from "components/CustomLink";
import TargetsInput from "components/TargetsInput";

import LabelForm from "../LabelForm";
import { ILabelFormData } from "../LabelForm/LabelForm";
import { generateTableHeaders } from "./LabelHostTargetTableConfig";

const baseClass = "ManualLabelForm";

export const LABEL_TARGET_HOSTS_INPUT_LABEL = "Select hosts";
const LABEL_TARGET_HOSTS_INPUT_PLACEHOLDER =
  "Search name, hostname, or serial number";
const DEBOUNCE_DELAY = 500;
const LABEL_YAML_DOCS_URL =
  "https://fleetdm.com/docs/configuration/yaml-files#labels";

export interface IManualLabelFormData {
  name: string;
  description: string;
  targetedHosts: IHost[];
}

interface ITargetsQueryKey {
  scope: string;
  query?: string | null;
  excludedHostIds?: number[];
}

interface IManualLabelFormProps {
  defaultName?: string;
  defaultDescription?: string;
  defaultTargetedHosts?: IHost[];
  isEditing?: boolean;
  teamName: string | null;
  onSave: (formData: IManualLabelFormData) => void;
  onCancel: () => void;
}

const ManualLabelForm = ({
  defaultName = "",
  defaultDescription = "",
  defaultTargetedHosts = [],
  isEditing = false,
  teamName,
  onSave,
  onCancel,
}: IManualLabelFormProps) => {
  const { gitOpsModeEnabled: labelsGitOpsManaged } = useGitOpsMode("labels");
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState("");
  const [isDebouncing, setIsDebouncing] = useState(false);
  const [targetedHosts, setTargetedHosts] = useState<IHost[]>(
    defaultTargetedHosts
  );

  const targetdHostsIds = targetedHosts.map((host) => host.id);

  const debounceSearch = useDebouncedCallback(
    (search: string) => {
      setDebouncedSearchQuery(search);
      setIsDebouncing(false);
    },
    DEBOUNCE_DELAY,
    { trailing: true }
  );

  // TODO: find a better way to debounce search requests
  useEffect(() => {
    setIsDebouncing(true);
    debounceSearch(searchQuery);
  }, [debounceSearch, searchQuery]);

  const {
    data: searchResults,
    isLoading: isLoadingSearchResults,
    isError: isErrorSearchResults,
  } = useQuery<ITargetsSearchResponse, Error, IHost[], ITargetsQueryKey[]>(
    [
      {
        scope: "labels-targets-search",
        query: debouncedSearchQuery,
        excludedHostIds: targetdHostsIds,
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
    setTargetedHosts((prevHosts) => prevHosts.concat(row.original));
    setSearchQuery("");
  };

  const onHostRemove = (row: Row<IHost>) => {
    setTargetedHosts((prevHosts) =>
      prevHosts.filter((h) => h.id !== row.original.id)
    );
  };

  const onSaveNewLabel = (
    labelFormData: ILabelFormData,
    labelFormDataValid: boolean
  ) => {
    if (labelFormDataValid) {
      // values from LabelForm component must be valid too
      onSave({ ...labelFormData, targetedHosts });
    }
  };

  const onChangeSearchQuery = (value: string) => {
    setSearchQuery(value);
  };

  const resultsTableConfig = generateTableHeaders();
  const selectedHostsTableConfig = generateTableHeaders(onHostRemove);

  // Only editing gets the split gate. Creating a label is a definition change, so GitOps mode
  // still locks the whole form there.
  const gitOpsLocksDefinitionOnly = isEditing;

  return (
    <div className={baseClass}>
      <LabelForm
        defaultName={defaultName}
        defaultDescription={defaultDescription}
        teamName={teamName}
        onCancel={onCancel}
        onSave={onSaveNewLabel}
        immutableFields={teamName ? ["fleets"] : []}
        gitOpsLocksDefinitionOnly={gitOpsLocksDefinitionOnly}
        additionalFields={
          <>
            <TargetsInput
              label={LABEL_TARGET_HOSTS_INPUT_LABEL}
              placeholder={LABEL_TARGET_HOSTS_INPUT_PLACEHOLDER}
              searchText={searchQuery}
              searchResultsTableConfig={resultsTableConfig}
              selectedHostsTableConifg={selectedHostsTableConfig}
              isTargetsLoading={isLoadingSearchResults || isDebouncing}
              hasFetchError={isErrorSearchResults}
              searchResults={searchResults ?? []}
              targetedHosts={targetedHosts}
              setSearchText={onChangeSearchQuery}
              handleRowSelect={onHostSelect}
            />
            {gitOpsLocksDefinitionOnly && labelsGitOpsManaged && (
              <span className="form-field__help-text">
                Omitting <b>hosts</b> in YAML preserves these hosts. Setting{" "}
                <b>hosts</b> in YAML replaces them on the next GitOps run.{" "}
                <CustomLink
                  newTab
                  text="Learn more"
                  url={LABEL_YAML_DOCS_URL}
                />
              </span>
            )}
          </>
        }
      />
    </div>
  );
};

export default ManualLabelForm;
