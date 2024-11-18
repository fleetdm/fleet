import React, {
  FormEvent,
  useCallback,
  useMemo,
  useState,
  useContext,
} from "react";

import mdmAppleBusinessManagerApi from "services/entities/mdm_apple_bm";
import { NotificationContext } from "context/notification";
import { getErrorReason } from "interfaces/errors";

import Icon from "components/Icon";
import Button from "components/buttons/Button";
import { downloadBase64ToFile, RequestState } from "./helpers";

interface IDownloadABMKeyProps {
  baseClass: string;
  onSuccess?: () => void;
  onError?: (e: unknown) => void;
}

const downloadKeyFile = (data: { public_key: string }) => {
  downloadBase64ToFile(data.public_key, "fleet-mdm-apple-bm-public-key.pem");
};

// TODO: why can't we use Content-Dispostion for these? We're only getting one file back now.

const useDownloadABMKey = ({
  onSuccess,
  onError,
}: Omit<IDownloadABMKeyProps, "baseClass">) => {
  const [downloadState, setDownloadState] = useState<RequestState>(undefined);
  const { renderFlash } = useContext(NotificationContext);

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
        const msg = getErrorReason(e);
        renderFlash("error", msg);
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
