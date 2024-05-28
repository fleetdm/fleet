import React, { FormEvent, useCallback, useMemo, useState } from "react";

import mdmAppleBusinessManagerApi from "services/entities/mdm_apple_bm";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import { downloadFile, RequestState } from "./helpers";

interface IDownloadABMKeyProps {
  baseClass: string;
  onSuccess?: () => void;
  onError?: (e: unknown) => void;
}

const downloadKeyFile = (data: { public_key: string }) => {
  downloadFile(data.public_key, "fleet-mdm-apple-bm-public-key.crt");
};

const useDownloadABMKey = ({
  onSuccess,
  onError,
}: Omit<IDownloadABMKeyProps, "baseClass">) => {
  const [downloadState, setDownloadState] = useState<RequestState>(undefined);

  const handleDownload = useCallback(
    async (evt: FormEvent) => {
      evt.preventDefault();
      setDownloadState("loading");
      try {
        const data = await mdmAppleBusinessManagerApi.downloadPublicKey();
        downloadKeyFile(data);
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

export const DownloadABMKey = ({
  baseClass,
  onSuccess,
  onError,
}: IDownloadABMKeyProps) => {
  const { handleDownload } = useDownloadABMKey({ onSuccess, onError });

  return (
    <Button
      className={`${baseClass}__request-button`}
      variant="text-icon"
      onClick={handleDownload}
    >
      <label htmlFor="download-key">
        <Icon name="download" color="core-fleet-blue" size="medium" />
        <span>Download public key</span>
      </label>
    </Button>
  );
};

export default DownloadABMKey;
