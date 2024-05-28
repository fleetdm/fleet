import React, { FormEvent, useCallback, useMemo, useState } from "react";

import mdmAppleApi from "services/entities/mdm_apple";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import { RequestState, downloadFile } from "./helpers";

interface IDownloadCSRProps {
  baseClass: string;
  onSuccess?: () => void;
  onError?: (e: unknown) => void;
}

const downloadCSRFile = (data: { csr: string }) => {
  downloadFile(data.csr, "fleet-mdm-apple.csr");
};

const useDownloadCSR = ({
  onSuccess,
  onError,
}: Omit<IDownloadCSRProps, "baseClass">) => {
  const [downloadState, setDownloadState] = useState<RequestState>(undefined);

  const handleDownload = useCallback(
    async (evt: FormEvent) => {
      evt.preventDefault();
      setDownloadState("loading");
      try {
        const data = await mdmAppleApi.requestCSR();
        downloadCSRFile(data);
        setDownloadState("success");
        onSuccess && onSuccess();
      } catch (e) {
        console.error(e);
        // TODO: error handling per Figma?
        setDownloadState("error");
        onError && onError(e);
      }
    },
    [onError, onSuccess]
  );

  const memoized = useMemo(
    () => ({
      downloadState,
      handleDownload,
    }),
    [downloadState, handleDownload]
  );

  return memoized;
};

export const DownloadCSR = ({
  baseClass,
  onSuccess,
  onError,
}: IDownloadCSRProps) => {
  const { handleDownload } = useDownloadCSR({ onSuccess, onError });

  return (
    <Button
      className={`${baseClass}__request-button`}
      variant="text-icon"
      onClick={handleDownload}
    >
      <label htmlFor="request-csr">
        <Icon name="download" color="core-fleet-blue" size="medium" />
        <span>Download CSR</span>
      </label>
    </Button>
  );
};

export default DownloadCSR;
