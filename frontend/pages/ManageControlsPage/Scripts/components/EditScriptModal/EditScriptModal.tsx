import { format } from "date-fns";
import FileSaver from "file-saver";
import React, {
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import {
  QueryObserverResult,
  RefetchOptions,
  RefetchQueryFilters,
  useQuery,
} from "react-query";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { IApiError, getErrorReason } from "interfaces/errors";
import { IHostScript } from "interfaces/script";
import scriptAPI, { IHostScriptsResponse } from "services/entities/scripts";

import ActionsDropdown from "components/ActionsDropdown";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import Editor from "components/Editor";
import Icon from "components/Icon";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Spinner from "components/Spinner";
import Textarea from "components/Textarea";
import { generateActionDropdownOptions } from "pages/hosts/details/HostDetailsPage/modals/RunScriptModal/ScriptsTableConfig";
import paths from "router/paths";

const baseClass = "edit-script-modal";

interface IEditScriptModal {
  onCancel: () => void;
  scriptId: number;
  scriptName: string;
  isHidden?: boolean;
  refetchHostScripts?: <TPageData>(
    options?: (RefetchOptions & RefetchQueryFilters<TPageData>) | undefined,
  ) => Promise<QueryObserverResult<IHostScriptsResponse, IApiError>>;
}

const EditScriptModal = ({ scriptId, scriptName, onCancel, isHidden, refetchHostScripts }: IEditScriptModal) => {
  // For scrollable modal
  const [isTopScrolling, setIsTopScrolling] = useState(false);
  const topDivRef = useRef<HTMLDivElement>(null);
  const checkScroll = () => {
    if (topDivRef.current) {
      const isScrolling =
        topDivRef.current.scrollHeight > topDivRef.current.clientHeight;
      setIsTopScrolling(isScrolling);
    }
  };

  const { renderFlash } = useContext(NotificationContext);

  const {
    data: scriptContent,
    error: isSelectedScriptContentError,
    isLoading: isLoadingSelectedScriptContent,
  } = useQuery<any, Error>(
    [scriptId],
    () => scriptAPI.downloadScript(scriptId),
    {
      refetchOnWindowFocus: false,
    },
  );

  // Editable script content
  const [scriptFormData, setScriptFormData] = useState("");
  useEffect(() => {
    setScriptFormData(scriptContent)
  }, [scriptContent]);

  // For scrollable modal
  useEffect(() => {
    checkScroll();
    window.addEventListener("resize", checkScroll);
    return () => window.removeEventListener("resize", checkScroll);
  }, [scriptContent, scriptFormData]); // Re-run when data changes

  const handleOnChange = (value: string) => {
    setScriptFormData(value);
  }

  return (
    <Modal
      className={baseClass}
      title={scriptName}
      width="large"
      onExit={onCancel}
      isHidden={isHidden}
    >
      <>
        <form>
          <Editor value={scriptFormData} onChange={handleOnChange}></Editor>
        </form>
        <ModalFooter
          isTopScrolling={isTopScrolling}
          secondaryButtons={
            <Button onClick={onCancel}>
              Cancel
            </Button>
          }
          primaryButtons={
              <Button onClick={onCancel} variant="brand">
                Done
              </Button>
          }
        />
      </>
    </Modal>
  )
};

export default EditScriptModal;
