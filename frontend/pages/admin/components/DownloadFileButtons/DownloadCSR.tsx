import React, { FormEvent, useCallback, useMemo, useState } from "react";

import mdmAppleApi from "services/entities/mdm_apple";
import pkiApi from "services/entities/pki";

import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";

import { RequestState, downloadBase64ToFile } from "./helpers";

// TODO: Refactor this interface to be more generalizable; probably some kind of fetchData
// callback that resolves to a base64 string and props for filename, button label, disable button,
// tooltip content, etc., and leave parms for the fetch (like pkiName) to the caller.
interface IDownloadCSRProps {
  baseClass: string;
  onSuccess?: () => void;
  onError?: (e: unknown) => void;
  pkiName?: string;
  disabled?: boolean;
}

const downloadCSRFile = (data: { csr: string }, filename?: string) => {
  downloadBase64ToFile(data.csr, filename || "fleet-mdm-apple.csr");
};

// TODO: why can't we use Content-Dispostion for these? We're only getting one file back now.

const useDownloadCSR = ({
  onSuccess,
  onError,
  pkiName,
}: Omit<IDownloadCSRProps, "baseClass">) => {
  const [downloadState, setDownloadState] = useState<RequestState>(undefined);

  const handleDownload = useCallback(
    async (evt: FormEvent) => {
      evt.preventDefault();
      setDownloadState("loading");
      try {
        let data;
        if (pkiName) {
          data = await pkiApi.requestCSR(pkiName);
        } else {
          data = await mdmAppleApi.requestCSR();
        }
        downloadCSRFile(data, pkiName);
        setDownloadState("success");
        onSuccess && onSuccess();
      } catch (e) {
        setDownloadState("error");
        onError && onError(e);
      }
    },
    [onError, onSuccess, pkiName]
  );

  const memoized = useMemo(
    () => ({
      downloadState,
      handleDownload,
      isLoading: downloadState === "loading",
    }),
    [downloadState, handleDownload]
  );

  return memoized;
};

export const DownloadCSR = ({
  baseClass,
  onSuccess,
  onError,
  pkiName,
  disabled = false,
}: IDownloadCSRProps) => {
  const { isLoading, handleDownload } = useDownloadCSR({
    onSuccess,
    onError,
    pkiName,
  });

  return (
    <TooltipWrapper
      tipContent="Complete all fields to download CSR"
      position="top"
      showArrow
      underline={false}
      tipOffset={-8}
      disableTooltip={!disabled || isLoading}
    >
      <Button
        className={`${baseClass}__request-button`}
        variant="text-icon"
        onClick={handleDownload}
        disabled={disabled || isLoading}
      >
        <label htmlFor="request-csr">
          <Icon name="download" color="core-fleet-blue" size="medium" />
          <span>Download CSR</span>
        </label>
      </Button>
    </TooltipWrapper>
  );
};

export default DownloadCSR;
