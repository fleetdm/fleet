import { useCallback, useState } from "react";

interface IUseToggleSidePanelHook {
  isSidePanelOpen: boolean;
  toggleSidePanel: () => void;
  setSidePanelOpen: (isOpen: boolean) => void;
}

const useToggleSidePanel = (
  initialIsOpened: boolean
): IUseToggleSidePanelHook => {
  const [isSidePanelOpen, setIsOpen] = useState<boolean>(initialIsOpened);

  const toggleSidePanel = useCallback(() => {
    setIsOpen(!isSidePanelOpen);
  }, [setIsOpen, isSidePanelOpen]);

  const setSidePanelOpen = useCallback(
    (isOpen: boolean) => {
      setIsOpen(isOpen);
    },
    [setIsOpen]
  );

  return {
    isSidePanelOpen,
    toggleSidePanel,
    setSidePanelOpen,
  };
};

export default useToggleSidePanel;
