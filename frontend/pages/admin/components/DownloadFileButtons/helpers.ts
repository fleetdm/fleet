export type RequestState = "loading" | "error" | "success" | undefined;

export const downloadBase64ToFile = (data: string, fileName: string) => {
  const linkSource = `data:application/octet-stream;base64,${data}`;
  const downloadLink = document.createElement("a");

  downloadLink.href = linkSource;
  downloadLink.download = fileName;
  downloadLink.click();
};
