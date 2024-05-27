import React, { FormEvent, useCallback, useContext, useState } from "react";

import { NotificationContext } from "context/notification";

import Icon from "components/Icon";
import Button from "components/buttons/Button";

export type RequestState = "loading" | "error" | "success" | undefined;

type RequestCsrResponse = {
  apns_key: string;
  scep_key: string;
  scep_cert: string;
};

type ResponseKeys = keyof RequestCsrResponse;

type CSRFile = {
  name: string;
  key: ResponseKeys;
  value?: string;
};

const FILES: CSRFile[] = [
  { name: "mdmcert.download.push.key", key: "apns_key" }, // APNS key
  { name: "fleet-mdm-apple-scep.key", key: "scep_key" }, // SCEP key
  { name: "fleet-mdm-apple-scep.crt", key: "scep_cert" }, // SCEP cert
];

const downloadFile = (tokens: string, fileName: string) => {
  const linkSource = `data:application/octet-stream;base64,${tokens}`;
  const downloadLink = document.createElement("a");

  downloadLink.href = linkSource;
  downloadLink.download = fileName;
  downloadLink.click();
};

const downloadCSRFiles = (data: RequestCsrResponse) => {
  // TODO: test this
  FILES.forEach((file) => {
    downloadFile(data[file.key], file.name);
  });
};

const useDownloadCSR = ({
  onSuccess,
  onError,
}: {
  onSuccess?: () => void;
  onError?: () => void;
}) => {
  const [downloadState, setDownloadState] = useState<RequestState>(undefined);

  const handleDownload = useCallback(
    async (evt: FormEvent) => {
      evt.preventDefault();
      setDownloadState("loading");
      try {
        // const data = await MdmAPI.requestCSR();
        // downloadCSRFiles(data);
        // setRequestState("success");
        console.log("Download CSR clicked");
        onSuccess && onSuccess();
      } catch (e) {
        console.error(e);
        // TODO: error handling per Figma
        setDownloadState("error");
        onError && onError();
      }
    },
    [onError, onSuccess]
  );

  return {
    downloadState,
    handleDownload,
  };
};

export const DownloadCSR = ({ baseClass }: { baseClass: string }) => {
  const { renderFlash } = useContext(NotificationContext);
  const { downloadState, handleDownload } = useDownloadCSR({});

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
