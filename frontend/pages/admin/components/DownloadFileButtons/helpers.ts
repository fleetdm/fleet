export type RequestState = "loading" | "error" | "success" | undefined;

export const downloadFile = (tokens: string, fileName: string) => {
  const linkSource = `data:application/octet-stream;base64,${tokens}`;
  const downloadLink = document.createElement("a");

  downloadLink.href = linkSource;
  downloadLink.download = fileName;
  downloadLink.click();
};
